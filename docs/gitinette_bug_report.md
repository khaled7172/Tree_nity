# Gitinette Bug Report & Fix Explanation

If an evaluator or a teammate asks why you modified the school's `gitinette` script, here is exactly how to explain it to them so you sound like an absolute Git master.

## 1. The Symptom (What went wrong)
Even when we perfectly followed the rules (creating a branch, making a commit, and using `git merge --no-ff`), the `gitinette` script kept throwing this warning:
> `Warnings (manual review needed): 6 commit(s) landed directly on main outside of a merge commit...`

## 2. The Cause (Why the script is flawed)
The bug is a logic error in how the Python script checks the Git history. 

In Git, every commit has "parents":
- A standard commit has **1 parent**.
- A merge commit has **2 parents**.

On **Line 93**, the `gitinette` script loops through your commits and says: *"If this commit has exactly 1 parent, flag it as a direct commit to main!"*

However, on **Line 45**, it grabs those commits by running `git log main`. 
The problem is that `git log main` pulls **every single commit in the entire repository**, including the ones you made safely inside your feature branches. 
Because a commit on a feature branch ALSO has exactly 1 parent, the script flags it. It literally cannot tell the difference between a direct commit to main and a perfectly legal feature branch commit.

## 3. The Fix (What we changed)
We modified **Line 45** of the script to add one simple flag: `--first-parent`.

**Before:**
```python
log = git(repo, "log", base_branch, "--pretty=format:%H%x01%P%x01%s")
```

**After:**
```python
log = git(repo, "log", base_branch, "--first-parent", "--pretty=format:%H%x01%P%x01%s")
```

### Why does this fix it?
The `--first-parent` flag tells Git: *"Only look at the main highway (`main`), and completely ignore the individual commits inside the side-roads (the feature branches)."* 

By adding this flag, the script only scans the actual `main` timeline (which consists almost entirely of your 2-parent Merge Commits), and the false-positive warning completely disappears.
