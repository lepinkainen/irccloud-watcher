# IRCCloud Watcher

This tool watches specified IRC channels on IRCCloud and generates a daily AI summary of discussions.

## Usage

1.  **Configure:**

    - Rename `config.yaml.example` to `config.yaml`.
    - Edit `config.yaml` with your IRCCloud API token and desired channels.

2.  **Run the watcher:**

    ```bash
    ./build/irccloud-watcher
    ```

3.  **Generate a summary on demand:**
    ```bash
    ./build/irccloud-watcher --generate-summary
    ```

## Development

- **Build:** `task build`
- **Test:** `task test`
- **Lint:** `task lint`
