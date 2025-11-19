"""BERTScore comparer for very large strings.

Usage examples:

	python bert_score/compare.py --candidate-file a.txt --reference-file b.txt
	python bert_score/compare.py --candidate-text "foo" --reference-text "bar"

"""

from __future__ import annotations

import argparse
import json
import sys
from dataclasses import dataclass
from itertools import zip_longest
from pathlib import Path
from statistics import mean
from typing import List, Sequence, Tuple

from bert_score import score as bertscore_score


def _positive_int(value: str, *, label: str) -> int:
	try:
		parsed = int(value)
	except ValueError as exc:
		raise argparse.ArgumentTypeError(f"{label} must be an integer") from exc
	if parsed <= 0:
		raise argparse.ArgumentTypeError(f"{label} must be > 0")
	return parsed


def parse_args() -> argparse.Namespace:
	parser = argparse.ArgumentParser(
		description=(
			"Compare two large pieces of text using BERTScore with optional chunking "
			"so you can inspect how similar they are."
		)
	)

	parser.add_argument("--candidate-text", help="Candidate text to compare")
	parser.add_argument(
		"--candidate-file",
		type=Path,
		help="Path to candidate text (use '-' to read from STDIN)",
	)
	parser.add_argument("--reference-text", help="Reference text to compare against")
	parser.add_argument(
		"--reference-file",
		type=Path,
		help="Path to reference text (use '-' to read from STDIN)",
	)
	parser.add_argument(
		"--encoding",
		default="utf-8",
		help="Encoding to use when reading files (default: utf-8)",
	)

	parser.add_argument(
		"--model",
		default="microsoft/deberta-base-mnli",
		help="Transformers model to use for scoring (default: microsoft/deberta-base-mnli)",
	)
	parser.add_argument(
		"--lang",
		default=None,
		help="ISO language code used by bert-score. Leave empty to let the library decide.",
	)
	parser.add_argument(
		"--batch-size",
		type=lambda v: _positive_int(v, label="batch-size"),
		default=8,
		help="Batch size for BERTScore (default: 8)",
	)
	parser.add_argument(
		"--chunk-size",
		type=lambda v: _positive_int(v, label="chunk-size"),
		default=350,
		help="Approximate number of words per chunk (default: 350)",
	)
	parser.add_argument(
		"--chunk-overlap",
		type=lambda v: _positive_int(v, label="chunk-overlap"),
		default=40,
		help="Number of overlapping words between chunks (default: 40)",
	)
	parser.add_argument(
		"--max-chunks",
		type=lambda v: _positive_int(v, label="max-chunks"),
		help="Maximum number of chunk pairs to score (useful for extremely long files)",
	)
	parser.add_argument(
		"--no-chunk",
		action="store_true",
		help="Disable chunking and compare the raw texts directly",
	)
	parser.add_argument(
		"--use-idf",
		action="store_true",
		help="Enable IDF weighting inside BERTScore (slower, but can be more accurate)",
	)
	parser.add_argument(
		"--rescale", action="store_true", help="Rescale scores using baseline statistics"
	)
	parser.add_argument("--lower", action="store_true", help="Lowercase text before scoring")
	parser.add_argument(
		"--normalize-whitespace",
		action="store_true",
		help="Collapse repeated whitespace characters before scoring",
	)
	parser.add_argument(
		"--show-chunks",
		type=lambda v: _positive_int(v, label="show-chunks"),
		default=5,
		help="Number of per-chunk rows to print (default: 5). Use 0 to hide chunk details.",
	)
	parser.add_argument(
		"--json",
		type=Path,
		help="Optional path to store the raw scores as JSON",
	)

	return parser.parse_args()


def load_text(text_value: str | None, file_value: Path | None, *, encoding: str, label: str) -> str:
	if text_value and file_value:
		raise SystemExit(f"Provide either --{label}-text or --{label}-file, not both.")
	if text_value:
		return text_value
	if file_value:
		if str(file_value) == "-":
			return sys.stdin.read()
		return file_value.read_text(encoding=encoding)
	raise SystemExit(f"You must provide either --{label}-text or --{label}-file.")


def normalize_text(text: str, *, lower: bool, collapse_whitespace: bool) -> str:
	cleaned = " ".join(text.split()) if collapse_whitespace else text
	return cleaned.lower() if lower else cleaned


def chunk_text(text: str, *, chunk_size: int, chunk_overlap: int) -> List[str]:
	words = text.split()
	if not words:
		return [""]

	effective_overlap = min(chunk_overlap, chunk_size - 1) if chunk_size > 1 else 0
	step = max(chunk_size - effective_overlap, 1)

	chunks: List[str] = []
	for start in range(0, len(words), step):
		chunk_words = words[start : start + chunk_size]
		chunks.append(" ".join(chunk_words))
	return chunks or [""]


def align_chunks(candidate_chunks: Sequence[str], reference_chunks: Sequence[str]) -> List[Tuple[str, str]]:
	return [
		(candidate or "", reference or "")
		for candidate, reference in zip_longest(candidate_chunks, reference_chunks, fillvalue="")
	]


@dataclass
class BertScoreResult:
	precision: List[float]
	recall: List[float]
	f1: List[float]


def run_bertscore(
	candidates: Sequence[str],
	references: Sequence[str],
	*,
	model: str,
	lang: str | None,
	batch_size: int,
	use_idf: bool,
	rescale: bool,
) -> BertScoreResult:
	precision_tensor, recall_tensor, f1_tensor = bertscore_score(
		candidates,
		references,
		model_type=model,
		lang=lang,
		batch_size=batch_size,
		rescale_with_baseline=rescale,
		idf=use_idf,
	)
	return BertScoreResult(
		precision=precision_tensor.tolist(),
		recall=recall_tensor.tolist(),
		f1=f1_tensor.tolist(),
	)


def summarize_scores(
	scores: BertScoreResult,
	chunk_pairs: Sequence[Tuple[str, str]],
) -> dict:
	token_weights = [
		max(len(candidate.split()), len(reference.split()), 1)
		for candidate, reference in chunk_pairs
	]
	total_weight = sum(token_weights)
	weighted_precision = sum(p * w for p, w in zip(scores.precision, token_weights)) / total_weight
	weighted_recall = sum(r * w for r, w in zip(scores.recall, token_weights)) / total_weight
	weighted_f1 = sum(f * w for f, w in zip(scores.f1, token_weights)) / total_weight

	chunk_details = [
		{
			"chunk": idx + 1,
			"precision": scores.precision[idx],
			"recall": scores.recall[idx],
			"f1": scores.f1[idx],
			"candidate_words": len(candidate.split()),
			"reference_words": len(reference.split()),
			"candidate_excerpt": candidate[:160],
			"reference_excerpt": reference[:160],
		}
		for idx, (candidate, reference) in enumerate(chunk_pairs)
	]

	return {
		"chunks": len(chunk_pairs),
		"precision": {
			"mean": mean(scores.precision),
			"weighted": weighted_precision,
			"min": min(scores.precision),
			"max": max(scores.precision),
		},
		"recall": {
			"mean": mean(scores.recall),
			"weighted": weighted_recall,
			"min": min(scores.recall),
			"max": max(scores.recall),
		},
		"f1": {
			"mean": mean(scores.f1),
			"weighted": weighted_f1,
			"min": min(scores.f1),
			"max": max(scores.f1),
		},
		"per_chunk": chunk_details,
	}


def print_summary(summary: dict, *, show_chunks: int) -> None:
	def _fmt(value: float) -> str:
		return f"{value:.4f}"

	print("BERTScore summary")
	print(f"  Precision  mean={_fmt(summary['precision']['mean'])}  weighted={_fmt(summary['precision']['weighted'])}")
	print(f"  Recall     mean={_fmt(summary['recall']['mean'])}  weighted={_fmt(summary['recall']['weighted'])}")
	print(f"  F1         mean={_fmt(summary['f1']['mean'])}  weighted={_fmt(summary['f1']['weighted'])}")
	print(f"  Chunks compared: {summary['chunks']}")

	if show_chunks <= 0:
		return

	print("\nWorst chunks by F1:")
	worst_chunks = sorted(summary["per_chunk"], key=lambda row: row["f1"])[:show_chunks]
	for row in worst_chunks:
		candidate_excerpt = row["candidate_excerpt"].replace("\n", " ")
		reference_excerpt = row["reference_excerpt"].replace("\n", " ")
		print(
			f"  #{row['chunk']:02d} F1={row['f1']:.4f} P={row['precision']:.4f} R={row['recall']:.4f} | "
			f"Cand: {candidate_excerpt[:120]!r} | Ref: {reference_excerpt[:120]!r}"
		)


def main() -> None:
	args = parse_args()
	encoding = args.encoding

	candidate_raw = load_text(
		args.candidate_text,
		args.candidate_file,
		encoding=encoding,
		label="candidate",
	)
	reference_raw = load_text(
		args.reference_text,
		args.reference_file,
		encoding=encoding,
		label="reference",
	)

	candidate_text = normalize_text(
		candidate_raw, lower=args.lower, collapse_whitespace=args.normalize_whitespace
	)
	reference_text = normalize_text(
		reference_raw, lower=args.lower, collapse_whitespace=args.normalize_whitespace
	)

	if args.no_chunk:
		candidate_chunks = [candidate_text]
		reference_chunks = [reference_text]
	else:
		candidate_chunks = chunk_text(
			candidate_text, chunk_size=args.chunk_size, chunk_overlap=args.chunk_overlap
		)
		reference_chunks = chunk_text(
			reference_text, chunk_size=args.chunk_size, chunk_overlap=args.chunk_overlap
		)

	chunk_pairs = align_chunks(candidate_chunks, reference_chunks)
	if args.max_chunks:
		chunk_pairs = chunk_pairs[: args.max_chunks]

	candidates = [candidate for candidate, _ in chunk_pairs]
	references = [reference for _, reference in chunk_pairs]

	scores = run_bertscore(
		candidates,
		references,
		model=args.model,
		lang=args.lang,
		batch_size=args.batch_size,
		use_idf=args.use_idf,
		rescale=args.rescale,
	)
	summary = summarize_scores(scores, chunk_pairs)
	print_summary(summary, show_chunks=args.show_chunks)

	if args.json:
		args.json.write_text(json.dumps(summary, indent=2))


if __name__ == "__main__":
	main()
 
