# Welcome to Your New Wails3 Project!

Congratulations on generating your Wails3 application! This README will guide you through the next steps to get your project up and running.

## Getting Started

1. Navigate to your project directory in the terminal.

2. To run your application in development mode, use the following command:

   ```
   wails3 dev
   ```

   This will start your application and enable hot-reloading for both frontend and backend changes.

3. To build your application for production, use:

   ```
   wails3 build
   ```

   This will create a production-ready executable in the `build` directory.

## Exploring Wails3 Features

Now that you have your project set up, it's time to explore the features that Wails3 offers:

1. **Check out the examples**: The best way to learn is by example. Visit the `examples` directory in the `v3/examples` directory to see various sample applications.

2. **Run an example**: To run any of the examples, navigate to the example's directory and use:

   ```
   go run .
   ```

   Note: Some examples may be under development during the alpha phase.

3. **Explore the documentation**: Visit the [Wails3 documentation](https://v3.wails.io/) for in-depth guides and API references.

4. **Join the community**: Have questions or want to share your progress? Join the [Wails Discord](https://discord.gg/JDdSxwjhGf) or visit the [Wails discussions on GitHub](https://github.com/wailsapp/wails/discussions).

## Project Structure

Take a moment to familiarize yourself with your project structure:

- `frontend/`: Contains your frontend code (HTML, CSS, JavaScript/TypeScript)
- `main.go`: The entry point of your Go backend
- `app.go`: Define your application structure and methods here
- `wails.json`: Configuration file for your Wails project

## Next Steps

1. Modify the frontend in the `frontend/` directory to create your desired UI.
2. Add backend functionality in `main.go`.
3. Use `wails3 dev` to see your changes in real-time.
4. When ready, build your application with `wails3 build`.

Happy coding with Wails3! If you encounter any issues or have questions, don't hesitate to consult the documentation or reach out to the Wails community.

## License

The main program (everything outside `core/`) is free software licensed under
the GNU General Public License, version 3 or (at your option) any later
version.

SPDX-License-Identifier: `GPL-3.0-or-later`

See [LICENSE](LICENSE) for the full GPL-3.0 text.

The `core/` directory is a separate Go module,
`github.com/link-fgfgui/mod-downloader/core`, licensed under the GNU Lesser
General Public License, version 3 or (at your option) any later version, with
the LGPL-3.0 Linking Exception. The legacy
`github.com/link-fgfgui/mod-downloader-core` module path is deprecated.

SPDX-License-Identifier: `LGPL-3.0-or-later WITH LGPL-3.0-linking-exception`

See [core/LICENSE](core/LICENSE) and [core/LICENSE.exception](core/LICENSE.exception).
Every Go source file declares its applicable SPDX license identifier in its
header.
