# Conventional Commits: Cheat Sheet

When you make a commit, it must look like this:
`type(scope): your description here`

Here is every single `type` allowed by the `gitinette` script, along with when you should use it:

| Type | Use Case / When to use it | Example |
| :--- | :--- | :--- |
| **`feat`** | **New Feature:** You added a new capability to the project. | `feat(server): add topic creation` |
| **`fix`** | **Bug Fix:** You fixed a bug or crash. | `fix(client): handle EOF on empty message` |
| **`chore`** | **Maintenance:** Routine tasks that don't change the source code (e.g., updating .gitignore, setting up the repo). | `chore(repo): initial commit` |
| **`refactor`** | **Code Rewrite:** Changing the code structure without changing its behavior (e.g., renaming variables, cleaning up functions). | `refactor(hashmap): simplify collision resolution loop` |
| **`test`** | **Testing:** Adding missing tests or correcting existing ones. | `test(prefix): add edge case tests for empty prefix` |
| **`docs`** | **Documentation:** Changes to the README, adding comments, or creating explanation files. | `docs(readme): add architecture section` |
| **`style`** | **Formatting:** Code style changes (spacing, commas, missing semicolons) that do not affect the logic. | `style(server): run gofmt on main.go` |
| **`perf`** | **Performance:** A code change that specifically improves performance. | `perf(hashmap): optimize memory allocation for new clients` |
| **`build`** | **Build System:** Changes that affect the build system or external dependencies (like your `Makefile`). | `build(makefile): add re and test rules` |
| **`ci`** | **Continuous Integration:** Changes to CI configuration files and scripts (e.g., GitHub Actions). *You probably won't use this much for this project.* | `ci(github): add workflow to test build on push` |
| **`revert`** | **Undo:** When you need to revert a previous commit. | `revert(client): undo previous connection timeout change` |

## What is the `(scope)`?
The scope is optional but highly recommended. It tells people *where* the change happened. 
For this project, good scopes would be: `(server)`, `(client)`, `(hashmap)`, `(ipc)`, `(readme)`, `(makefile)`.
