export const HarnessHooks = async ({ directory, $ }) => {
  const sessionFile = `${directory}/.opencode/harness-session.json`;
  const pluginRoot = `${directory}/.opencode/plugins`;

  return {
    "shell.env": async (_input, output) => {
      output.env.HARNESS_SESSION_FILE = sessionFile;
      output.env.HARNESS_PLUGIN_ROOT = pluginRoot;
    },
    "tool.execute.before": async (input, output) => {
      if (input.tool === "bash") {
        const command = output?.args?.command || "";

        if (/(^|\s)grep(\s|$)/.test(command) && !/(^|\s)rg(\s|$)/.test(command)) {
          throw new Error("Use rg instead of grep in OpenCode workflows.");
        }

        if (/rm\s+-rf\s+\//.test(command)) {
          throw new Error("Refusing dangerous rm -rf / command.");
        }
      }

      if (input.tool === "read" && output?.args?.filePath && output?.args?.filePath.includes(".env")) {
        throw new Error("Do not read .env files directly.");
      }
    },
    "tool.execute.after": async (input, output) => {
      if (input.tool === "write" || input.tool === "edit") {
        const path = output?.args?.filePath || output?.args?.path;
        if (path && path.endsWith(".md")) {
          // Keep markdown edits visible in the session state for debugging.
          await $`printf '%s\n' ${path.replace(/'/g, "'\\''")} >> ${sessionFile}`;
        }
      }
    },
    "experimental.session.compacting": async (_input, output) => {
      output.context.push(`
## Harness Session

OpenCode session state:
- Session file: ${sessionFile}
- Plugin root: ${pluginRoot}
`);
    },
    event: async ({ event }) => {
      if (event.type === "session.idle") {
        await $`osascript -e 'display notification "OpenCode session idle" with title "harness-hooks"'`;
      }
    },
  };
};
