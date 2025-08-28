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
		<div className="flex flex-col items-center justify-center min-h-screen bg-gray-100 p-8">
			<img src={logo} alt="logo" className="w-32 h-32 mb-8" />
			<div className="text-xl text-gray-800 mb-8 text-center">{resultText}</div>
			<div className="flex gap-4">
				<input
					className="px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
					onChange={updateName}
					autoComplete="off"
					name="input"
					type="text"
					placeholder="Enter your name"
				/>
				<button
					type="button"
					className="px-6 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
					onClick={greet}
				>
					Greet
				</button>
			</div>
		</div>
	);
}

export default App;
