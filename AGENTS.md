# Repository guidance

## Communication

Use ASD-STE100 Simplified Technical English for all text you write: questions, updates, explanations, final answers, commit messages, PR descriptions, and review comments. Be as concise as possible. Keep quotations, code, commands, paths, identifiers, and required technical terms unchanged.

When a step doesn't need the developer's input, keep going. Put status notes in the same message as your next action. Stop and ask only when you can't continue without the developer, or before anything destructive: deleting data, force-pushing, or changing anything outside this repository.

For work with more than a few steps, keep a checklist in `TASKS.md` at the workspace root (VCS ignores it). Tick each item when it is done, and add new items that you find. The file stays available after the context is summarized.

For an audit or migration that divides into independent areas, give each area to its own subagent.
Check the evidence in each subagent's report before you accept it. Finish with one table.

Mark each claim that you could not confirm, and say where you looked.

**Non-negotiable** rules protect user data, released contracts, and security. Change them only on an explicit developer decision. All other rules are defaults: follow the developer's preference and state which default you set aside. Do not argue by citing this file.
## Non-negotiable

## Oracle

- `.refs/CLIProxyAPI` is the main source
- `.refs/BumpyClock-Cli-Proxy-API-Management-Center` is the source and reference for the frontend (`apps/web`)
- `.refs/CPA-Manager-Plus` is ideas for the dashboard
- `.refs/magpie` is for some ideas and features
- https://gist.github.com/t3dotgg/59a521d2ee8d70aaf90e280c175072af

## Workflow (sub-agents)

You work on exactly one issue at a time, in your own workspace under `.workspaces/`, on the bookmark/branch `issue/NNNN-slug`. The coordinator creates the workspace for you.

1. Read the issue. Set `status: in-progress` in its header and commit that.
2. Work in small steps and **commit often** (VCS change/commit on your bookmark/branch. Message format is Conventional Commits, plus a body when the reason is not obvious.
3. Keep the issue file's `## Log` section current: what you did, what you found, open questions. This file is how context survives between agents and sessions.
4. If you make a design decision that others must know about, append it to `DECISIONS.md` on your bookmark/branch (format given in that file).
5. If you discover follow-up work, list it under `## Follow-ups` in your issue. Do not create new issue files; the coordinator does that.
6. When the acceptance criteria are met:
    - set `status: done` in the issue header and fill in `## Result` (what landed, how to use it, known gaps),
    - commit,
    - merge to main from your workspace. It rebases onto `main`, rebasing and gating again if `main` moved meanwhile. If the rebase conflicts, resolve it and run the script again. If the gate fails, fix the failure; never weaken the gate to get green.
7. Report back to the coordinator with a short summary: what landed, the commit range,follow-ups, anything surprising.
