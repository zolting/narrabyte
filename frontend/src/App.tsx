import { useState } from "react";
import { Greet } from "../wailsjs/go/main/App";
import logo from "./assets/images/logo-universal.png";

function App() {
	const [resultText, setResultText] = useState(
		"Please enter your name below 👇",
	);
	const [name, setName] = useState("");
	const updateName = (e: any) => setName(e.target.value);
	const updateResultText = (result: string) => setResultText(result);

	function greet() {
		Greet(name).then(updateResultText);
	}

	return (
		<div className="flex min-h-screen flex-col items-center justify-center bg-gray-100 p-8">
			<img alt="logo" className="mb-8 h-32 w-32" src={logo} />
			<div className="mb-8 text-center text-gray-800 text-xl">{resultText}</div>
			<div className="flex gap-4">
				<input
					autoComplete="off"
					className="rounded-lg border border-gray-300 px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
					name="input"
					onChange={updateName}
					placeholder="Enter your name"
					type="text"
				/>
				<button
					className="rounded-lg bg-blue-500 px-6 py-2 text-white transition-colors hover:bg-blue-600"
					onClick={greet}
					type="button"
				>
					Greet
				</button>
			</div>
		</div>
	);
}

export default App;
