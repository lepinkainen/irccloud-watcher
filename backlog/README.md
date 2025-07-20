# LLM Task Management Instructions

---

## template: backlog_readme

This directory (`@backlog/`) contains individual task files for your project. Each task’s status is encoded in its filename.

**Naming Convention:**
`[task-id]-[short-description]--[status].md`

- `[task-id]`: A numerical prefix (e.g., `01`, `02`) used for ordering and unique identification.
- `[short-description]`: A concise, hyphen-separated description of the task.
- `[status]`: The current status of the task.

**Task Statuses:**

- **`--todo` (or no suffix):** The task is identified and ready to be started. Files without a status suffix are implicitly `--todo`.
  - _LLM Action:_ Identify these as the next tasks to work on.
- **`--in-progress`:** The LLM is currently working on this task.
  - _LLM Action:_ Rename a `--todo` task to `--in-progress` when beginning work on it.
- **`--blocked`:** The task cannot proceed because it's waiting for external input (e.g., user clarification, a dependency to be resolved).
  - _LLM Action:_ Rename an `--in-progress` task to `--blocked` if progress is halted.
- **`--review`:** The LLM has completed its work, but human review or approval is required.
  - _LLM Action:_ Rename an `--in-progress` task to `--review` when work is complete and ready for human validation.
- **`--done`:** The task is fully finished and verified.
  - _LLM Action:_ Rename an `--in-progress` or `--review` task to `--done` once all criteria are met and verified.
- **`--skipped` / `--cancelled`:** The task has been explicitly decided not to be implemented, or it's no longer relevant.
  - _LLM Action:_ Rename any task to `--skipped` or `--cancelled` if instructed by the user or if it becomes obsolete.

## LLM Workflow for Task Selection and Management:

1.  **Discover Tasks**: Use `list_directory` or `glob` on `@backlog/` to get an overview of all task files.
2.  **Prioritize**: Focus on files that do _not_ have `--done`, `--skipped`, or `--cancelled` suffixes. Prefer tasks with `--todo` (or no suffix) or `--in-progress` status.
3.  **Start Task**: When beginning work on a new task, rename its file from `[task-id]-[short-description]--todo.md` (or no suffix) to `[task-id]-[short-description]--in-progress.md`.
4.  **Execute Task**: Follow the instructions within the task file.
5.  **Complete Task**: Upon successful completion and verification, rename the task file to `[task-id]-[short-description]--done.md`.
6.  **Handle Blocks**: If a task becomes blocked, rename it to `[task-id]-[short-description]--blocked.md` and inform the user.

## Progress Tracking

All IRCCloud Watcher authentication and WebSocket connection issues have been resolved!

| File                                    | Description                          | Status |
| :-------------------------------------- | :----------------------------------- | :----- |
| 01-fix-authentication--done.md         | Fix authentication headers and flow  | done   |
| 02-dynamic-websocket--done.md          | Implement dynamic WebSocket URLs     | done   |
| 03-enhance-structures--done.md         | Update JSON response structures      | done   |
| 04-add-tests--done.md                  | Add comprehensive tests              | done   |
| 05-integration--done.md                | Final integration and validation     | done   |

**✅ ALL TASKS COMPLETED** - The IRCCloud Watcher is now fully functional!
