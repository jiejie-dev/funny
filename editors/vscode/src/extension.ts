import * as vscode from "vscode";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;

interface PlanGraphNode {
  id: string;
  label: string;
  kind: string;
  parentId?: string;
  timeout?: string;
  retry?: { max: number; backoff?: string; on?: string[] };
}

interface PlanGraphEdge {
  from: string;
  to: string;
  kind: string;
}

interface PlanGraph {
  name: string;
  nodes: PlanGraphNode[];
  edges: PlanGraphEdge[];
}

interface PlanGraphResult {
  plans: PlanGraph[];
}

export function activate(context: vscode.ExtensionContext): void {
  const config = vscode.workspace.getConfiguration("funny");
  const serverPath = config.get<string>("lsp.path") ?? "funny";
  const serverArgs = config.get<string[]>("lsp.args") ?? ["lsp"];

  const serverOptions: ServerOptions = {
    run: {
      command: serverPath,
      args: serverArgs,
      transport: TransportKind.stdio,
    },
    debug: {
      command: serverPath,
      args: serverArgs,
      transport: TransportKind.stdio,
    },
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "funny" }],
    synchronize: {
      fileEvents: vscode.workspace.createFileSystemWatcher("**/*.{fn,funny}"),
    },
  };

  client = new LanguageClient(
    "funnyLanguageServer",
    "Funny Language Server",
    serverOptions,
    clientOptions
  );

  context.subscriptions.push(
    client,
    vscode.commands.registerCommand("funny.runFile", runCurrentFile),
    vscode.commands.registerCommand("funny.formatFile", formatCurrentFile),
    vscode.commands.registerCommand("funny.showPlanGraph", showPlanGraph),
    vscode.commands.registerCommand(
      "funny.restartLanguageServer",
      restartLanguageServer
    )
  );

  void client.start();
}

export async function deactivate(): Promise<void> {
  if (client) {
    await client.stop();
  }
}

async function runCurrentFile(): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor || editor.document.languageId !== "funny") {
    vscode.window.showWarningMessage("Open a Funny (.fn / .funny) file first.");
    return;
  }

  const funnyPath =
    vscode.workspace.getConfiguration("funny").get<string>("executablePath") ??
    "funny";
  const filePath = editor.document.uri.fsPath;
  const cwd = vscode.workspace.getWorkspaceFolder(editor.document.uri)?.uri.fsPath;

  const terminal = vscode.window.createTerminal({
    name: "Funny Run",
    cwd,
  });
  terminal.show();
  terminal.sendText(`${quoteShell(funnyPath)} run ${quoteShell(filePath)}`);
}

async function formatCurrentFile(): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor || editor.document.languageId !== "funny") {
    vscode.window.showWarningMessage("Open a Funny (.fn / .funny) file first.");
    return;
  }
  await vscode.commands.executeCommand("editor.action.formatDocument");
}

async function restartLanguageServer(): Promise<void> {
  if (!client) {
    return;
  }
  await client.stop();
  await client.start();
  vscode.window.showInformationMessage("Funny language server restarted.");
}

async function showPlanGraph(): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor || editor.document.languageId !== "funny") {
    vscode.window.showWarningMessage("Open a Funny (.fn / .funny) file first.");
    return;
  }
  if (!client) {
    vscode.window.showErrorMessage("Funny language server is not running.");
    return;
  }

  const uri = editor.document.uri.toString();
  let result: PlanGraphResult;
  try {
    result = await client.sendRequest<PlanGraphResult>("funny/planGraph", {
      textDocument: { uri },
    });
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    vscode.window.showErrorMessage(`Plan graph request failed: ${message}`);
    return;
  }

  if (!result.plans || result.plans.length === 0) {
    vscode.window.showInformationMessage("No plan blocks found in this file.");
    return;
  }

  const panel = vscode.window.createWebviewPanel(
    "funnyPlanGraph",
    `Plan: ${result.plans.map((p) => p.name).join(", ")}`,
    vscode.ViewColumn.Beside,
    { enableScripts: true }
  );
  panel.webview.html = renderPlanGraphHtml(result.plans);
}

function renderPlanGraphHtml(plans: PlanGraph[]): string {
  const sections = plans
    .map((plan) => {
      const mermaid = planToMermaid(plan);
      return `
        <section>
          <h2>${escapeHtml(plan.name)}</h2>
          <pre class="mermaid">${escapeHtml(mermaid)}</pre>
          <details>
            <summary>Nodes (${plan.nodes.length})</summary>
            <ul>${plan.nodes
              .map(
                (n) =>
                  `<li><code>${escapeHtml(n.id)}</code> — ${escapeHtml(n.label)} <span class="kind">${escapeHtml(n.kind)}</span></li>`
              )
              .join("")}</ul>
          </details>
        </section>`;
    })
    .join("\n");

  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <style>
    body { font-family: var(--vscode-font-family); color: var(--vscode-foreground); background: var(--vscode-editor-background); padding: 1rem; }
    h2 { margin-top: 0; border-bottom: 1px solid var(--vscode-panel-border); padding-bottom: 0.5rem; }
    section { margin-bottom: 2rem; }
    .kind { opacity: 0.7; font-size: 0.85em; }
    details { margin-top: 1rem; }
    pre.mermaid { background: var(--vscode-textBlockQuote-background); padding: 1rem; border-radius: 6px; overflow-x: auto; white-space: pre-wrap; }
  </style>
  <script type="module">
    import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
    mermaid.initialize({ startOnLoad: true, theme: 'dark' });
  </script>
</head>
<body>
  ${sections}
</body>
</html>`;
}

function planToMermaid(plan: PlanGraph): string {
  const lines: string[] = ["flowchart TD"];
  const idMap = new Map<string, string>();

  for (const node of plan.nodes) {
    const safeId = sanitizeMermaidId(node.id);
    idMap.set(node.id, safeId);
    const label = `${node.label}\\n(${node.kind})`;
    lines.push(`  ${safeId}["${label}"]`);
  }

  for (const edge of plan.edges) {
    const from = idMap.get(edge.from) ?? sanitizeMermaidId(edge.from);
    const to = idMap.get(edge.to) ?? sanitizeMermaidId(edge.to);
    const style =
      edge.kind === "parallel" ? "-- parallel -->" : "-- sequence -->";
    lines.push(`  ${from} ${style} ${to}`);
  }

  return lines.join("\n");
}

function sanitizeMermaidId(id: string): string {
  return id.replace(/[^a-zA-Z0-9_]/g, "_");
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function quoteShell(value: string): string {
  if (/^[a-zA-Z0-9_./-]+$/.test(value)) {
    return value;
  }
  return `"${value.replace(/"/g, '\\"')}"`;
}
