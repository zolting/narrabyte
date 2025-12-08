# README

## About

This is the official Narrabyte project.

You can install the executable application inside the Releases view and simply running the file.

## Live Development

To run in live development mode, run `wails dev` in the project directory. This will run a Vite development
server that will provide very fast hot reload of your frontend changes. If you want to develop in a browser
and have access to your Go methods, there is also a dev server that runs on http://localhost:34115. Connect
to this in your browser, and you can call your Go code from devtools.

## Building

To build a redistributable, production mode package, use `wails build`.

## Updating

To update the official release, create a tag and release for the new version and pipeline will automatically create a build for Windows, Ubuntu and MacOS.
