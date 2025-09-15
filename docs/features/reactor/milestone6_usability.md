# **Feature Design Document: M6 - Usability & Workflow Polish**

Version: 1.3
Status: Approved
Author(s): Gemini, cam
Date: 2025-09-02

## **1. The 'Why': Rationale & User Focus**
*(This section remains unchanged)*

## **2. The 'How': Technical Design & Architecture**

This milestone involves targeted fixes across several packages, primarily `cmd/reactor/` and `pkg/docker`.

### **Task 1: Improve `accounts list` and Add `accounts clean`**

*   **Problem:** Output is cryptic; no cleanup mechanism exists.
*   **Detailed Design:**
    1.  **Modify `orchestrator.Up()`:** This function in `pkg/orchestrator/orchestrator.go` must be modified. After a container is successfully provisioned, it will write a file named `project-path.txt` into the project-specific config directory (`~/.reactor/{account}/{hash}/`). This file will contain the absolute path of the project directory.
    2.  **Modify `accountsListHandler`:** The handler for `reactor accounts list` will be updated. It will now read the `project-path.txt` file from within each project-hash directory and display it next to the hash in the format: `  - /path/to/project (hash)`.
    3.  **New Command `accounts clean`:**
        *   Add a new subcommand: `reactor accounts clean`.
        *   The handler will scan all account and project-hash directories in `~/.reactor/`.
        *   For each, it will read the `project-path.txt` and check if the path still exists. If not, it is considered "orphaned".
        *   It will list all orphaned directories and **prompt the user for confirmation** before deleting them. Deletion should not be the default.

### **Task 2: Make `config set/get` Useful**

*   **Problem:** Commands are unhelpful, only telling the user to edit a file.
*   **Detailed Design:**
    1.  **Scope:** The `config set/get` commands will **only operate on known keys** within the `customizations.reactor` block (e.g., `account`, `defaultCommand`). The command should return an error if a user tries to set an unsupported key.
    2.  **Modify `configSetHandler`:**
        *   This handler must be reworked to read, modify, and write to the `devcontainer.json` file.
        *   **Critical Requirement:** The modification **must preserve comments and existing file formatting.**
        *   **Library:** Use a library like `github.com/tidwall/sjson` to surgically update the JSON without reformatting the entire file.
        *   The logic must be able to create `customizations.reactor` if it doesn't exist and add/update the specified key.
    3.  **Modify `configGetHandler`:**
        *   This handler will now read and parse the `devcontainer.json` file and print the actual value of the requested key.
        *   If a requested key does not exist, the command should print an empty string and exit with code 0.

### **Task 3: Fix CLI Typos and Missing Commands**

*   **Problem:** `sessions list` help text is wrong; `workspace` commands are missing.
*   **Detailed Design:**
    1.  **`sessionsListHandler`:** In `cmd/reactor/main.go`, find the `fmt.Println` statement and change `reactor run` to `reactor up`.
    2.  **`newRootCmd`:** In `cmd/reactor/main.go`, ensure that `cmd.AddCommand(newWorkspaceCmd())` is present and not commented out.

### **Task 4: Improve Container Detach Experience**

*   **Problem:** Detaching is slow and shows a "broken pipe" error. The `up` command calls a non-existent function.
*   **Detailed Design:**
    1.  **Modify `ExecuteInteractiveCommand`:** This function in `pkg/docker/service.go` is the correct location for the fix.
    2.  **Add Parameter:** The function signature should be changed to accept a new boolean parameter, `isInteractive`.
    3.  **Enable Detach Keys:** When `isInteractive` is `true`, the `types.ContainerAttachOptions` struct passed to the Docker client must have `DetachKeys` set to `"ctrl-p,ctrl-q"`.
    4.  **Suppress Error:** The error handling for the `stdin` copy operation must specifically check if the error string contains "write: broken pipe". If it does, the function should return `nil`, as this is an expected and normal outcome.
    5.  **Update Callers:** The `upCmdHandler` must be updated to call `ExecuteInteractiveCommand` (passing the default shell as the command) instead of the non-existent `AttachInteractiveSession`.

### **Task 5: Polish CLI Output and Help Text**

*   **Problem:** `exec` help is confusing, `list` outputs are inconsistent, and the CLI lacks color.
*   **Detailed Design:**
    1.  **`exec` Help:** Update the `Long` help text for `reactor exec` and `reactor workspace exec` to emphasize the use of `--`. Change the primary example to `reactor exec -- ls -la /tmp`.
    2.  **Standardize `list` Tables:** Create a new `pkg/ui` package with a function for rendering tables. Use this shared function in both `sessions list` and `workspace list` to ensure their columns (`SERVICE`/`NAME`, `STATUS`, `IMAGE`, `PROJECT PATH`) and formatting are identical.
    3.  **Add Color:** Add the `github.com/fatih/color` library. Apply colors consistently: green for success, yellow for warnings, red for errors. For `workspace up`, use a predefined, sequential list of colors for each service's log prefix.
    4.  **Consistent `--verbose`:** The `up`, `down`, `build`, and all `workspace` subcommands must be updated to respect the `--verbose` flag by printing the **full, marshalled `ResolvedConfig` struct** before execution.

## **3. The 'What': Implementation & Execution**

### **3.1. Phased implementation plan**

This work can be done as a single, comprehensive PR focusing on usability.

*   [ ] **Task 1:** Implement the `accounts list` and `accounts clean` functionality.
*   [ ] **Task 2:** Rework the `config set` and `config get` commands to directly interact with `devcontainer.json`.
*   [ ] **Task 3:** Fix the `sessions list` help text and ensure the `workspace` command is registered.
*   [ ] **Task 4:** Improve the container detach experience by modifying `ExecuteInteractiveCommand`.
*   [ ] **Task 5:** Implement all CLI output and help text polishing tasks.
*   [ ] **Task 6:** Add/update integration tests for all the above fixes to prevent regressions.

### **3.2. Testing strategy**

*   **`accounts list/clean` test:** An integration test must create an orphaned config, run `accounts clean`, and verify that the orphaned config is correctly identified and can be removed.
*   **`config set` test:** An integration test must take a `devcontainer.json` file with comments, run `reactor config set`, and assert that the file was modified correctly *and* that the original comments and formatting are preserved.
*   **Manual Verification:** The detach behavior (`Ctrl+P,Ctrl+Q`) and the visual appeal of colored output will require manual testing and verification.