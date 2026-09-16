# Git Workflow Guidelines Cheat Sheet

To pass the `gitinette` evaluation, follow these strict rules for every change.

## 1. Branching
**Never commit directly to `main`** (except for tiny, non-breaking typos). Always create a branch.
- **Rule:** Branch name must match the commit type (e.g., `<type>/<feature-name>`).
- **Good:** `feat/ipc-server`, `fix/memory-leak`, `chore/makefile`
- **Bad:** `my-feature`, `test-branch`

## 2. Commit Messages (Conventional Commits)
Every commit must follow this format: `type(scope): description`
- **Allowed Types:** `feat`, `fix`, `chore`, `refactor`, `test`, `docs`, `style`, `perf`, `build`, `ci`, `revert`
- **Rule:** One commit = one logical change.
- **Example:** `feat(client): add prefix filtering logic`

## 3. Updating Your Branch (DO NOT MERGE MAIN)
If `main` has new code that you need in your feature branch, **do not** run `git merge main`. 
- **Rule:** You must rebase to keep history linear.
```bash
git fetch
git rebase main
```

## 4. Merging to Main (NO FAST-FORWARD)
When your feature is complete, merge it into `main` using the `--no-ff` flag. This creates a dedicated merge commit, which is required.
```bash
git checkout main
git merge --no-ff feat/my-feature
```
*Note: Squashing commits during the merge is NOT allowed. The evaluators want to see your intermediate commits.*

## 5. Fixing Mistakes Before Merging
If you made a typo in a commit message or forgot a file, use interactive rebase on your branch before merging it to `main`.
```bash
# Edit the last 3 commits
git rebase -i HEAD~3
```

## 6. How to check your work
Run the provided python script to verify your history:
```bash
# Standard check
./gitinette

# Evaluation check (samples commits to make sure messages match code)
./gitinette --evaluation
```

 I am now focusing on explicitly mapping out all relevant tools before executing any commands. My latest initiative is the creation of a `guidelines.md` file to store these tools. I have formul...
  I've saved all those rules into guidelines.md so you can keep them open as a cheat sheet while you work.

  To answer your questions:
  ### 1. Running gitinette

  Yes, you run it from the terminal in your project folder using:
    ./gitinette

  (Note: It's ./ not /.)
  If it gives you a "permission denied" error, you just need to make it executable first by running chmod +x gitinette. Alternatively, since it's a Python script, you can always run it by typing
  python3 gitinette.

  ### 2. Working on GitHub vs. Vogsphere

  Yes, you can absolutely work on GitHub first. In fact, it's highly recommended so you and your team can use Pull Requests and code reviews.

  HOWEVER, there is one massive trap you must avoid:
  When you are done, DO NOT just copy and paste your final files into a fresh Vogsphere folder and make a single "final project" commit. If you do that, you will lose all your branches and commit
  history, gitinette will instantly fail you, and you will get a 0.

  The right way to do it:
  You need to push your entire Git history from GitHub directly to Vogsphere. You do this by adding Vogsphere as a second "remote" to your existing local repository.

  When it's time to submit, you will run something like this:

    # Add vogsphere as a remote destination
    git remote add vogsphere <your-vogsphere-repo-url>

    # Push your main branch AND your full commit history to vogsphere
    git push vogsphere main

  By doing it this way, Vogsphere receives every single commit, branch, and merge exactly as you did them on GitHub, keeping the evaluators and the gitinette script happy!
  