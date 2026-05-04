#!/usr/bin/env bun

import { mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { basename, dirname, join } from "node:path";
import { homedir, platform } from "node:os";
import { createInterface } from "node:readline/promises";
import { stdin as input, stdout as output } from "node:process";
import { handleVpsPlansCommand } from "./commands/vpsPlans";
import { getVpsPlanChoiceHint, isVpsPlanOrderable } from "./vps/availability";

const DEFAULT_BASE_URL = "https://dritestudio.co.th";
const API_PREFIX = "/api/auth";
const METHODS = new Set(["GET", "POST", "PUT", "PATCH", "DELETE"]);
const DURATIONS = new Set(["daily", "weekly", "monthly", "yearly"]);

type JsonValue = null | boolean | number | string | JsonValue[] | { [key: string]: JsonValue };

type Config = {
  baseUrl?: string;
  apiKey?: string;
};

type ParsedArgs = {
  positionals: string[];
  flags: Map<string, string[]>;
};

type RequestOptions = {
  method?: string;
  path: string;
  body?: unknown;
  form?: FormData;
  query?: Record<string, string>;
  token?: string;
  baseUrl?: string;
};

type ApiResult = {
  ok: boolean;
  status: number;
  data: unknown;
};

type Choice<T> = {
  label: string;
  value: T;
  hint?: string;
};

class CliError extends Error {
  code: number;

  constructor(message: string, code = 1) {
    super(message);
    this.code = code;
  }
}

function configPath() {
  if (platform() === "win32" && process.env.APPDATA) {
    return join(process.env.APPDATA, "drite", "config.json");
  }

  return join(process.env.XDG_CONFIG_HOME || join(homedir(), ".config"), "drite", "config.json");
}

async function loadConfig(): Promise<Config> {
  try {
    const raw = await readFile(configPath(), "utf8");
    return JSON.parse(raw) as Config;
  } catch {
    return {};
  }
}

async function saveConfig(config: Config) {
  const file = configPath();
  await mkdir(dirname(file), { recursive: true });
  await writeFile(file, `${JSON.stringify(config, null, 2)}\n`, { mode: 0o600 });
}

function parseArgs(argv: string[]): ParsedArgs {
  const positionals: string[] = [];
  const flags = new Map<string, string[]>();

  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];

    if (!arg.startsWith("--")) {
      positionals.push(arg);
      continue;
    }

    const raw = arg.slice(2);
    const eq = raw.indexOf("=");
    const key = eq >= 0 ? raw.slice(0, eq) : raw;
    const inline = eq >= 0 ? raw.slice(eq + 1) : undefined;
    const next = argv[i + 1];
    const value = inline ?? (next && !next.startsWith("--") ? next : "true");

    if (inline === undefined && next && !next.startsWith("--")) {
      i += 1;
    }

    const existing = flags.get(key) ?? [];
    existing.push(value);
    flags.set(key, existing);
  }

  return { positionals, flags };
}

function emptyArgs(): ParsedArgs {
  return {
    positionals: [],
    flags: new Map()
  };
}

function flag(args: ParsedArgs, name: string, fallback?: string): string | undefined {
  return args.flags.get(name)?.at(-1) ?? fallback;
}

function flagAll(args: ParsedArgs, name: string): string[] {
  return args.flags.get(name) ?? [];
}

function boolFlag(args: ParsedArgs, name: string): boolean {
  return args.flags.has(name) && flag(args, name) !== "false";
}

async function parseJsonFlag(args: ParsedArgs): Promise<unknown | undefined> {
  const jsonFile = flag(args, "json-file");
  let raw = flag(args, "json") ?? flag(args, "data");

  if (jsonFile) {
    raw = await readFile(jsonFile, "utf8");
  } else if (raw?.startsWith("@")) {
    raw = await readFile(raw.slice(1), "utf8");
  }

  if (!raw) return undefined;

  try {
    return JSON.parse(raw) as JsonValue;
  } catch (error: any) {
    throw new CliError(`Invalid JSON passed to --json: ${error.message}`);
  }
}

function parseQuery(args: ParsedArgs): Record<string, string> {
  const query: Record<string, string> = {};

  for (const item of flagAll(args, "query")) {
    const eq = item.indexOf("=");
    if (eq <= 0) throw new CliError(`Invalid --query value "${item}". Use key=value.`);
    query[item.slice(0, eq)] = item.slice(eq + 1);
  }

  for (const name of ["page", "limit", "status", "month", "domain"]) {
    const value = flag(args, name);
    if (value !== undefined) query[name] = value;
  }

  return query;
}

function requireArg(value: string | undefined, name: string): string {
  if (!value) throw new CliError(`Missing required argument: ${name}`);
  return value;
}

function requireFlag(args: ParsedArgs, name: string): string {
  return requireArg(flag(args, name), `--${name}`);
}

function normalizeBaseUrl(url: string) {
  return url.replace(/\/+$/, "");
}

function normalizeApiPath(path: string) {
  const normalized = path.startsWith("/") ? path : `/${path}`;
  if (normalized.startsWith("/api/")) return normalized;
  return normalized.startsWith(API_PREFIX) ? normalized : `${API_PREFIX}${normalized}`;
}

function withQuery(path: string, query: Record<string, string> = {}) {
  const entries = Object.entries(query).filter(([, value]) => value !== "");
  if (entries.length === 0) return path;

  const params = new URLSearchParams(entries);
  return `${path}${path.includes("?") ? "&" : "?"}${params.toString()}`;
}

async function resolveAuth(args: ParsedArgs) {
  const config = await loadConfig();
  const token = flag(args, "token") ?? process.env.DRITE_API_KEY ?? config.apiKey;
  const baseUrl = normalizeBaseUrl(flag(args, "base-url") ?? process.env.DRITE_API_URL ?? config.baseUrl ?? DEFAULT_BASE_URL);

  return { token, baseUrl, config };
}

async function requestJson(args: ParsedArgs, options: RequestOptions): Promise<ApiResult> {
  const auth = await resolveAuth(args);
  const token = options.token ?? auth.token;
  if (!token) {
    throw new CliError("No API token found. Run `drite auth login --token <token>` or set DRITE_API_KEY.");
  }

  const baseUrl = normalizeBaseUrl(options.baseUrl ?? auth.baseUrl);
  const path = withQuery(normalizeApiPath(options.path), options.query);
  const headers: Record<string, string> = {
    Authorization: `Bearer ${token}`,
    Accept: "application/json"
  };

  const init: RequestInit = {
    method: options.method ?? "GET",
    headers
  };

  if (options.form) {
    init.body = options.form;
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    init.body = JSON.stringify(options.body);
  }

  const response = await fetch(`${baseUrl}${path}`, init);
  const contentType = response.headers.get("content-type") ?? "";
  const text = await response.text();
  const data = contentType.includes("application/json") && text ? JSON.parse(text) : text;

  return {
    ok: response.ok,
    status: response.status,
    data
  };
}

async function apiRequest(args: ParsedArgs, options: RequestOptions) {
  const result = await requestJson(args, options);

  if (!result.ok) {
    printOutput({ ok: false, status: result.status, data: result.data }, boolFlag(args, "compact"));
    process.exit(result.status >= 400 && result.status < 600 ? 1 : 0);
  }

  printOutput(result.data, boolFlag(args, "compact"));
}

async function apiRequestAndMaybeWait(args: ParsedArgs, options: RequestOptions) {
  const result = await requestJson(args, options);

  if (!result.ok) {
    printOutput({ ok: false, status: result.status, data: result.data }, boolFlag(args, "compact"));
    process.exit(result.status >= 400 && result.status < 600 ? 1 : 0);
  }

  printOutput(result.data, boolFlag(args, "compact"));

  if (!boolFlag(args, "wait")) return;

  const jobId = extractJobId(result.data);
  if (!jobId) {
    console.log("No jobId returned; nothing to wait for.");
    return;
  }

  await waitForVpsJob(args, jobId);
}

async function interactiveRequest(options: RequestOptions) {
  const args = emptyArgs();
  const result = await requestJson(args, options);
  if (!result.ok) {
    printOutput({ ok: false, status: result.status, data: result.data }, false);
    return null;
  }

  printOutput(result.data, false);
  return result.data;
}

async function buildFormData(args: ParsedArgs): Promise<FormData | undefined> {
  const form = new FormData();
  let hasAny = false;

  for (const pair of flagAll(args, "field")) {
    const eq = pair.indexOf("=");
    if (eq <= 0) throw new CliError(`Invalid --field value "${pair}". Use key=value.`);
    form.append(pair.slice(0, eq), pair.slice(eq + 1));
    hasAny = true;
  }

  for (const pair of flagAll(args, "file")) {
    const eq = pair.indexOf("=");
    const field = eq > 0 ? pair.slice(0, eq) : "file";
    const filePath = eq > 0 ? pair.slice(eq + 1) : pair;
    const bytes = await readFile(filePath);
    const file = new File([bytes], basename(filePath), { type: guessMimeType(filePath) });
    form.append(field, file);
    hasAny = true;
  }

  return hasAny ? form : undefined;
}

function guessMimeType(filePath: string) {
  const lower = filePath.toLowerCase();
  if (lower.endsWith(".jpg") || lower.endsWith(".jpeg")) return "image/jpeg";
  if (lower.endsWith(".png")) return "image/png";
  if (lower.endsWith(".gif")) return "image/gif";
  if (lower.endsWith(".webp")) return "image/webp";
  if (lower.endsWith(".pdf")) return "application/pdf";
  return "application/octet-stream";
}

function printOutput(data: unknown, compact: boolean) {
  if (typeof data === "string") {
    console.log(data);
    return;
  }

  console.log(JSON.stringify(data, null, compact ? 0 : 2));
}

function sleep(ms: number) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

function asRecord(value: unknown): Record<string, any> {
  return value && typeof value === "object" && !Array.isArray(value) ? value as Record<string, any> : {};
}

function getString(value: unknown, fallback = "") {
  return typeof value === "string" && value.length > 0 ? value : fallback;
}

function numberFlag(args: ParsedArgs, name: string, fallback: number) {
  const value = flag(args, name);
  if (value === undefined) return fallback;
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    throw new CliError(`--${name} must be a positive number`);
  }
  return parsed;
}

function extractJobId(data: unknown): string | null {
  const root = asRecord(data);
  const nested = asRecord(root.data);
  return getString(root.jobId) || getString(nested.jobId) || getString(root.id) || null;
}

function isTerminalJobStatus(status: string) {
  return ["success", "failed", "completed", "error"].includes(status.toLowerCase());
}

async function waitForVpsJob(args: ParsedArgs, jobId: string) {
  const intervalMs = numberFlag(args, "interval", 3) * 1000;
  const timeoutMs = numberFlag(args, "timeout", 900) * 1000;
  const startedAt = Date.now();

  console.log(`Waiting for job ${jobId}...`);

  while (Date.now() - startedAt <= timeoutMs) {
    const result = await requestJson(args, { path: `/vps/job/${jobId}` });
    if (!result.ok) {
      printOutput({ ok: false, status: result.status, data: result.data }, boolFlag(args, "compact"));
      process.exit(1);
    }

    const payload = asRecord(asRecord(result.data).data ?? result.data);
    const status = getString(payload.status, "unknown");
    const name = getString(payload.name, "job");
    const attempts = payload.attempts !== undefined ? ` attempts=${payload.attempts}` : "";
    const queueLength = payload.queueLength !== undefined ? ` queue=${payload.queueLength}` : "";
    const error = getString(payload.error);

    console.log(`[${new Date().toLocaleTimeString()}] ${name} ${status}${attempts}${queueLength}${error ? ` error=${error}` : ""}`);

    if (isTerminalJobStatus(status)) {
      printOutput(result.data, boolFlag(args, "compact"));
      if (status.toLowerCase() !== "success" && status.toLowerCase() !== "completed") {
        process.exit(1);
      }
      return;
    }

    await sleep(intervalMs);
  }

  throw new CliError(`Timed out waiting for job ${jobId}`);
}

function booleanFromFlag(args: ParsedArgs, name: string): boolean {
  const value = requireFlag(args, name).toLowerCase();
  if (["true", "1", "yes", "on"].includes(value)) return true;
  if (["false", "0", "no", "off"].includes(value)) return false;
  throw new CliError(`--${name} must be true or false`);
}

function durationType(args: ParsedArgs) {
  const value = requireFlag(args, "duration");
  if (!DURATIONS.has(value)) {
    throw new CliError("--duration must be one of daily, weekly, monthly, yearly");
  }
  return value;
}

async function handleAuth(args: ParsedArgs, action: string | undefined) {
  const config = await loadConfig();

  if (action === "login" || action === "token") {
    const token = requireFlag(args, "token");
    await saveConfig({ ...config, apiKey: token, baseUrl: flag(args, "base-url") ?? config.baseUrl ?? DEFAULT_BASE_URL });
    console.log(`Token saved to ${configPath()}`);
    return;
  }

  if (action === "logout") {
    await rm(configPath(), { force: true });
    console.log("Token removed.");
    return;
  }

  if (action === "status") {
    const token = process.env.DRITE_API_KEY ?? config.apiKey;
    printOutput({
      baseUrl: process.env.DRITE_API_URL ?? config.baseUrl ?? DEFAULT_BASE_URL,
      tokenSource: process.env.DRITE_API_KEY ? "DRITE_API_KEY" : config.apiKey ? configPath() : null,
      authenticated: Boolean(token)
    }, false);
    return;
  }

  throw new CliError("Usage: drite auth login --token <token> | status | logout");
}

async function handleConfig(args: ParsedArgs, action: string | undefined) {
  const config = await loadConfig();

  if (action === "set-url") {
    const baseUrl = normalizeBaseUrl(requireArg(args.positionals[2], "baseUrl"));
    await saveConfig({ ...config, baseUrl });
    console.log(`Base URL saved: ${baseUrl}`);
    return;
  }

  if (action === "show" || !action) {
    printOutput({ baseUrl: config.baseUrl ?? DEFAULT_BASE_URL, configPath: configPath() }, false);
    return;
  }

  throw new CliError("Usage: drite config show | set-url <baseUrl>");
}

async function watchVps(args: ParsedArgs, id: string) {
  const intervalMs = numberFlag(args, "interval", 5) * 1000;
  const maxIterations = flag(args, "max") ? numberFlag(args, "max", 0) : 0;
  let iteration = 0;

  console.log(`Watching VPS ${id}. Press Ctrl+C to stop.`);

  while (maxIterations === 0 || iteration < maxIterations) {
    const [detail, status, activity] = await Promise.all([
      requestJson(args, { path: `/vps/${id}` }),
      requestJson(args, { path: `/vps/${id}/status` }),
      requestJson(args, { path: `/vps/${id}/activity` })
    ]);

    if (!detail.ok) {
      printOutput({ ok: false, status: detail.status, data: detail.data }, boolFlag(args, "compact"));
      process.exit(1);
    }

    const vps = asRecord(asRecord(detail.data).data ?? detail.data);
    const live = asRecord(asRecord(status.data).data ?? status.data);
    const jobs = asArray(asRecord(activity.data).data);
    const latestJob = asRecord(jobs.at(0));

    console.log(JSON.stringify({
      time: new Date().toISOString(),
      id: vps.id ?? id,
      name: vps.name,
      ip: vps.ip,
      status: vps.status,
      locked: vps.locked,
      percent: vps.percent,
      powerState: live.powerState,
      latestJob: latestJob.jobId ? {
        jobId: latestJob.jobId,
        name: latestJob.name,
        status: latestJob.status,
        attempts: latestJob.attempts,
        error: latestJob.error
      } : null
    }, null, boolFlag(args, "compact") ? 0 : 2));

    iteration += 1;
    if (maxIterations !== 0 && iteration >= maxIterations) return;
    await sleep(intervalMs);
  }
}

async function handleVps(args: ParsedArgs, action: string | undefined) {
  const id = args.positionals[2];

  switch (action) {
    case "list":
      return apiRequest(args, { path: "/vps", query: parseQuery(args) });
    case "plans":
      return handleVpsPlansCommand({
        args,
        compact: boolFlag(args, "compact"),
        query: parseQuery(args),
        templateId: flag(args, "template-id"),
        availableOnly: boolFlag(args, "available-only"),
        requestJson,
        printOutput
      });
    case "templates":
      return apiRequest(args, { path: "/vps/templates" });
    case "available-ips":
      return apiRequest(args, { path: `/vps/available-ips/${requireArg(id, "hostId")}` });
    case "get":
      return apiRequest(args, { path: `/vps/${requireArg(id, "id")}` });
    case "stats":
      return apiRequest(args, { path: `/vps/${requireArg(id, "id")}/stats` });
    case "status":
      return apiRequest(args, { path: `/vps/${requireArg(id, "id")}/status` });
    case "activity":
      return apiRequest(args, { path: `/vps/${requireArg(id, "id")}/activity` });
    case "upgrade-options":
      return apiRequest(args, { path: `/vps/${requireArg(id, "id")}/upgrade-options` });
    case "watch":
      return watchVps(args, requireArg(id, "id"));
    case "job":
      return apiRequest(args, { path: `/vps/job/${requireArg(id, "jobId")}` });
    case "failed":
      return apiRequest(args, { path: "/vps/failed" });
    case "ack-failed":
      return apiRequest(args, { method: "DELETE", path: `/vps/failed/${requireArg(id, "id")}` });
    case "create":
      return apiRequestAndMaybeWait(args, { method: "POST", path: "/vps", body: await parseJsonFlag(args) ?? {
        name: requireFlag(args, "name"),
        templateId: requireFlag(args, "template-id"),
        planId: requireFlag(args, "plan-id"),
        durationType: durationType(args),
        password: requireFlag(args, "password"),
        ip: flag(args, "ip"),
        networkRef: flag(args, "network-ref")
      } });
    case "renew":
      return apiRequest(args, { method: "POST", path: `/vps/${requireArg(id, "id")}/renew`, body: { durationType: durationType(args) } });
    case "upgrade":
      return apiRequestAndMaybeWait(args, { method: "POST", path: `/vps/${requireArg(id, "id")}/upgrade`, body: { planId: requireFlag(args, "plan-id") } });
    case "rename":
      return apiRequest(args, { method: "POST", path: `/vps/${requireArg(id, "id")}/rename`, body: { name: requireFlag(args, "name") } });
    case "auto-renew":
      return apiRequest(args, { method: "POST", path: `/vps/${requireArg(id, "id")}/auto-renewal`, body: { enabled: booleanFromFlag(args, "enabled") } });
    case "reinstall":
      return apiRequestAndMaybeWait(args, { method: "POST", path: `/vps/${requireArg(id, "id")}/reinstall`, body: await parseJsonFlag(args) ?? {
        templateId: requireFlag(args, "template-id"),
        password: requireFlag(args, "password")
      } });
    case "start":
    case "stop":
    case "reboot":
    case "force-stop":
      return apiRequestAndMaybeWait(args, { method: "POST", path: `/vps/${requireArg(id, "id")}/${action}` });
    case "control":
      return apiRequestAndMaybeWait(args, { method: "POST", path: `/vps/${requireArg(id, "id")}/control`, body: { action: requireFlag(args, "action") } });
    case "network-reset":
      return apiRequestAndMaybeWait(args, { method: "POST", path: `/vps/${requireArg(id, "id")}/network-reset` });
    case "reset-password":
      return apiRequestAndMaybeWait(args, { method: "POST", path: `/vps/${requireArg(id, "id")}/reset-password`, body: { password: requireFlag(args, "password") } });
    case "delete":
      return apiRequestAndMaybeWait(args, { method: "DELETE", path: `/vps/${requireArg(id, "id")}` });
    default:
      throw new CliError("Usage: drite vps list|get|watch|create|plans|templates|stats|status|start|stop|reboot|renew|upgrade|rename|delete ...");
  }
}

async function handleHosting(args: ParsedArgs, action: string | undefined) {
  const id = args.positionals[2];

  switch (action) {
    case "list":
      return apiRequest(args, { path: "/hosting/list" });
    case "plans":
      return apiRequest(args, { path: "/api/un_auth/hosting/plans" });
    case "check-domain":
      return apiRequest(args, { path: "/hosting/check-domain", query: { domain: requireArg(id ?? flag(args, "domain"), "domain") } });
    case "get":
      return apiRequest(args, { path: `/hosting/${requireArg(id, "id")}` });
    case "stats":
      return apiRequest(args, { path: `/hosting/${requireArg(id, "id")}/stats` });
    case "disk":
      return apiRequest(args, { path: `/hosting/${requireArg(id, "id")}/disk` });
    case "traffic":
      return apiRequest(args, { path: `/hosting/${requireArg(id, "id")}/traffic` });
    case "activity":
      return apiRequest(args, { path: `/hosting/${requireArg(id, "id")}/activity` });
    case "activation-status":
      return apiRequest(args, { path: `/hosting/${requireArg(id, "id")}/activation-status` });
    case "deploy":
      return apiRequest(args, { method: "POST", path: "/hosting/deploy", body: await parseJsonFlag(args) ?? {
        duration: Number(requireFlag(args, "duration-days")),
        domain: requireFlag(args, "domain"),
        planId: requireFlag(args, "plan-id"),
        password: requireFlag(args, "password")
      } });
    case "access":
      return apiRequest(args, { method: "POST", path: `/hosting/${requireArg(id, "id")}/access` });
    case "renew":
      return apiRequest(args, { method: "POST", path: `/hosting/${requireArg(id, "id")}/renew`, body: flag(args, "duration") ? { durationType: durationType(args) } : {} });
    case "autorenew":
      return apiRequest(args, { method: "POST", path: `/hosting/${requireArg(id, "id")}/autorenew` });
    case "reset-password":
      return apiRequest(args, { method: "POST", path: `/hosting/${requireArg(id, "id")}/reset-password`, body: { password: requireFlag(args, "password") } });
    case "delete":
      return apiRequest(args, { method: "DELETE", path: `/hosting/${requireArg(id, "id")}` });
    default:
      throw new CliError("Usage: drite hosting list|get|deploy|check-domain|stats|renew|access|delete ...");
  }
}

async function handleBilling(args: ParsedArgs, action: string | undefined) {
  switch (action) {
    case "transactions":
      return apiRequest(args, { path: "/transactions", query: parseQuery(args) });
    case "export":
      return apiRequest(args, { path: "/transactions/export", query: parseQuery(args) });
    case "topup-history":
      return apiRequest(args, { path: "/topup/history" });
    case "topup-status":
      return apiRequest(args, { path: `/topup/status/${requireArg(args.positionals[2], "referenceNo")}` });
    default:
      throw new CliError("Usage: drite billing transactions|export|topup-history|topup-status");
  }
}

async function handleTicket(args: ParsedArgs, action: string | undefined) {
  const id = args.positionals[2];

  switch (action) {
    case "list":
      return apiRequest(args, { path: "/ticket/list", query: parseQuery(args) });
    case "get":
      return apiRequest(args, { path: `/ticket/${requireArg(id, "id")}` });
    case "create":
      return apiRequest(args, { method: "POST", path: "/ticket", body: await parseJsonFlag(args) ?? {
        subject: requireFlag(args, "subject"),
        category: requireFlag(args, "category"),
        priority: flag(args, "priority", "normal"),
        message: requireFlag(args, "message"),
        serviceType: flag(args, "service-type"),
        serviceId: flag(args, "service-id")
      } });
    case "reply":
      return apiRequest(args, { method: "POST", path: `/ticket/${requireArg(id, "id")}/reply`, body: await parseJsonFlag(args) ?? { message: requireFlag(args, "message") } });
    case "close":
      return apiRequest(args, { method: "PUT", path: `/ticket/${requireArg(id, "id")}/close` });
    case "upload-url":
      return apiRequest(args, { method: "POST", path: "/ticket/upload-url", body: {
        filename: requireFlag(args, "filename"),
        mimeType: requireFlag(args, "mime-type")
      } });
    case "upload": {
      const form = await buildFormData(args);
      if (!form) throw new CliError("Usage: drite ticket upload --file <path>");
      return apiRequest(args, { method: "POST", path: "/ticket/upload", form });
    }
    default:
      throw new CliError("Usage: drite ticket list|get|create|reply|close|upload-url|upload ...");
  }
}

async function handleWebhook(args: ParsedArgs, action: string | undefined) {
  const id = args.positionals[2];

  switch (action) {
    case "list":
      return apiRequest(args, { path: "/me/webhooks" });
    case "create":
      return apiRequest(args, { method: "POST", path: "/me/webhooks", body: await parseJsonFlag(args) ?? {
        url: requireFlag(args, "url"),
        events: flagAll(args, "event")
      } });
    case "delete":
      return apiRequest(args, { method: "DELETE", path: `/me/webhooks/${requireArg(id, "id")}` });
    default:
      throw new CliError("Usage: drite webhook list|create|delete ...");
  }
}

async function handleApiKey(args: ParsedArgs, action: string | undefined) {
  switch (action) {
    case "status": {
      const result = await requestJson(args, { path: "/me" });
      if (!result.ok) {
        printOutput({ ok: false, status: result.status, data: result.data }, boolFlag(args, "compact"));
        process.exit(1);
      }

      const profile = asRecord(result.data);
      printOutput({
        userId: profile.id,
        email: profile.email,
        hasApiKey: Boolean(profile.hasApiKey),
        apiKeyPreview: profile.apiKeyPreview ?? null,
        apiAllowedIps: asArray(profile.apiAllowedIps),
        keyModel: "one active API key per user"
      }, boolFlag(args, "compact"));
      return;
    }
    case "create":
      return apiRequest(args, { method: "POST", path: "/me/api-key" });
    case "revoke":
      return apiRequest(args, { method: "DELETE", path: "/me/api-key" });
    case "security":
      return apiRequest(args, { method: "PUT", path: "/me/api-key/security", body: {
        allowedIps: flagAll(args, "ip")
      } });
    case "logs":
      return apiRequest(args, { path: "/me/api-logs", query: parseQuery(args) });
    default:
      throw new CliError("Usage: drite api-key status|create|revoke|security|logs ...");
  }
}

async function handleRaw(args: ParsedArgs) {
  const method = requireArg(args.positionals[1], "method").toUpperCase();
  if (!METHODS.has(method)) throw new CliError(`Unsupported HTTP method: ${method}`);

  return apiRequest(args, {
    method,
    path: requireArg(args.positionals[2], "path"),
    body: await parseJsonFlag(args),
    form: await buildFormData(args),
    query: parseQuery(args)
  });
}

async function handleDoctor(args: ParsedArgs) {
  const auth = await resolveAuth(args);
  const checks: Array<{ name: string; ok: boolean; detail: string }> = [];

  checks.push({
    name: "config",
    ok: Boolean(auth.baseUrl),
    detail: auth.baseUrl
  });

  checks.push({
    name: "token",
    ok: Boolean(auth.token),
    detail: auth.token ? "configured" : "missing"
  });

  if (!auth.token) {
    printOutput({ ok: false, checks }, boolFlag(args, "compact"));
    process.exit(1);
  }

  try {
    const profileResult = await requestJson(args, { path: "/me" });
    const profile = asRecord(profileResult.data);
    checks.push({
      name: "auth",
      ok: profileResult.ok,
      detail: profileResult.ok ? `${profile.email ?? profile.id ?? "authenticated"}` : `HTTP ${profileResult.status}`
    });
    checks.push({
      name: "api-key",
      ok: Boolean(profile.hasApiKey),
      detail: profile.hasApiKey ? `present (${profile.apiKeyPreview ?? "masked"})` : "not created for this user"
    });
  } catch (error: any) {
    checks.push({
      name: "auth",
      ok: false,
      detail: error?.message ?? String(error)
    });
  }

  try {
    const plansResult = await requestJson(args, { path: "/vps/plans" });
    checks.push({
      name: "vps-api",
      ok: plansResult.ok,
      detail: plansResult.ok ? "reachable" : `HTTP ${plansResult.status}`
    });
  } catch (error: any) {
    checks.push({
      name: "vps-api",
      ok: false,
      detail: error?.message ?? String(error)
    });
  }

  const ok = checks.every(check => check.ok || check.name === "api-key");
  printOutput({ ok, checks }, boolFlag(args, "compact"));
  if (!ok) process.exit(1);
}

function printHelp() {
  console.log(`Drite CLI

Usage:
  drite
  drite interactive
  drite auth login --token <dr_live_token>
  drite auth status
  drite config set-url <baseUrl>
  drite me
  drite doctor
  drite vps list|get|watch|create|plans|templates|stats|status|start|stop|reboot|force-stop|renew|upgrade-options|upgrade|rename|auto-renew|reinstall|network-reset|reset-password|delete
  drite hosting plans|list|get|deploy|check-domain|stats|disk|traffic|activity|activation-status|access|renew|autorenew|reset-password|delete
  drite billing transactions|export|topup-history|topup-status
  drite ticket list|get|create|reply|close|upload-url|upload
  drite webhook list|create|delete
  drite api-key status|create|revoke|security|logs
  drite raw <METHOD> <PATH> [--json '{...}'] [--query key=value] [--file field=path]
  drite plans all|hosting

Global flags:
  --token <token>       Use token for this call without saving it.
  --base-url <url>      Override API base URL.
  --compact            Print compact JSON.
  --wait               Poll returned VPS jobId until it finishes.
  --interval <seconds> Poll interval for --wait or watch. Default: 3 for --wait, 5 for watch.
  --timeout <seconds>  Timeout for --wait. Default: 900.

Examples:
  drite me
  drite doctor
  drite vps start <id> --wait
  drite vps plans --template-id <template_id>
  drite vps plans --available-only
  drite vps watch <id>
  drite vps renew <id> --duration monthly
  drite vps upgrade-options <id>
  drite vps upgrade <id> --plan-id <plan_id> --wait
  drite vps reset-password <id> --password "NewStrongPassw0rd!" --wait
  drite raw GET /api/auth/me
`);
}

function asArray(value: unknown): any[] {
  return Array.isArray(value) ? value : [];
}

function getPayloadList(data: any, key: string) {
  return asArray(data?.[key]);
}

async function withReadline<T>(handler: (rl: ReturnType<typeof createInterface>) => Promise<T>) {
  const rl = createInterface({ input, output });
  try {
    return await handler(rl);
  } finally {
    rl.close();
  }
}

async function ask(rl: ReturnType<typeof createInterface>, question: string, fallback = "") {
  const suffix = fallback ? ` (${fallback})` : "";
  const answer = (await rl.question(`${question}${suffix}: `)).trim();
  return answer || fallback;
}

async function confirm(rl: ReturnType<typeof createInterface>, question: string) {
  const answer = (await rl.question(`${question} [y/N]: `)).trim().toLowerCase();
  return answer === "y" || answer === "yes";
}

async function select<T>(rl: ReturnType<typeof createInterface>, title: string, choices: Choice<T>[]): Promise<T | null> {
  if (choices.length === 0) {
    console.log("No items available.");
    return null;
  }

  console.log(`\n${title}`);
  choices.forEach((choice, index) => {
    const hint = choice.hint ? ` - ${choice.hint}` : "";
    console.log(`  ${index + 1}. ${choice.label}${hint}`);
  });
  console.log("  0. Back");

  while (true) {
    const raw = await rl.question("Select: ");
    const index = Number(raw.trim());
    if (index === 0) return null;
    if (Number.isInteger(index) && index >= 1 && index <= choices.length) {
      return choices[index - 1].value;
    }
    console.log("Invalid selection.");
  }
}

async function pause(rl: ReturnType<typeof createInterface>) {
  await rl.question("\nPress Enter to continue...");
}

async function fetchInteractive(options: RequestOptions) {
  const result = await requestJson(emptyArgs(), options);
  if (!result.ok) {
    printOutput({ ok: false, status: result.status, data: result.data }, false);
    return null;
  }
  return result.data as any;
}

async function runAndShow(rl: ReturnType<typeof createInterface>, options: RequestOptions) {
  const data = await fetchInteractive(options);
  if (data !== null) {
    printOutput(data, false);
    const jobId = extractJobId(data);
    if (jobId && await confirm(rl, `Wait for job ${jobId}?`)) {
      await waitForVpsJob(emptyArgs(), jobId);
    }
  }
  await pause(rl);
  return data;
}

async function ensureInteractiveAuth(rl: ReturnType<typeof createInterface>) {
  const { token, config } = await resolveAuth(emptyArgs());
  if (token) return;

  console.log("No API token configured.");
  const apiKey = await ask(rl, "Paste API token");
  if (!apiKey) throw new CliError("API token is required.");
  await saveConfig({ ...config, apiKey, baseUrl: config.baseUrl ?? DEFAULT_BASE_URL });
  console.log(`Token saved to ${configPath()}`);
}

function pricedDurationChoices(prices: Record<string, any> | null | undefined): Choice<string>[] {
  return [
    { label: `Daily${formatPriceHint(prices?.dailyPrice)}`, value: "daily" },
    { label: `Weekly${formatPriceHint(prices?.weeklyPrice)}`, value: "weekly" },
    { label: `Monthly${formatPriceHint(prices?.monthlyPrice)}`, value: "monthly" },
    { label: `Yearly${formatPriceHint(prices?.yearlyPrice)}`, value: "yearly" }
  ];
}

function pricedHostingDurationChoices(prices: Record<string, any> | null | undefined): Choice<number>[] {
  return [
    { label: `1 day${formatPriceHint(prices?.dailyPrice)}`, value: 1 },
    { label: `7 days${formatPriceHint(prices?.weeklyPrice)}`, value: 7 },
    { label: `30 days${formatPriceHint(prices?.monthlyPrice)}`, value: 30 },
    { label: `365 days${formatPriceHint(prices?.yearlyPrice)}`, value: 365 }
  ];
}

function formatPriceHint(price: unknown) {
  if (price === null || price === undefined || price === "") return "";
  return ` - ${price} THB`;
}

function formatTemplateLabel(template: any) {
  const os = [template.os, template.version].filter(Boolean).join(" ").trim();
  return os || template.name || template.id;
}

async function selectVps(rl: ReturnType<typeof createInterface>) {
  const data = await fetchInteractive({ path: "/vps" });
  const items = getPayloadList(data, "data");
  return select(rl, "Select VPS", items.map(vps => ({
    label: `${vps.name || vps.id} (${vps.ip || "no-ip"})`,
    hint: `${vps.status || "unknown"} / ${vps.id}`,
    value: vps
  })));
}

async function selectHosting(rl: ReturnType<typeof createInterface>) {
  const data = await fetchInteractive({ path: "/hosting/list" });
  const items = getPayloadList(data, "active");
  return select(rl, "Select Hosting", items.map(hosting => ({
    label: `${hosting.name || hosting.id}`,
    hint: `${hosting.renew_at || "no-renew-date"} / ${hosting.id}`,
    value: hosting
  })));
}

async function interactiveVpsCreate(rl: ReturnType<typeof createInterface>) {
  const templatesData = await fetchInteractive({ path: "/vps/templates" });
  if (!templatesData) return;

  const template = await select(rl, "Select template", getPayloadList(templatesData, "data").map(template => ({
    label: formatTemplateLabel(template),
    hint: template.id,
    value: template
  })));
  if (!template) return;

  const plansData = await fetchInteractive({ path: "/vps/plans", query: { templateId: template.id } });
  if (!plansData) return;

  const availablePlans = getPayloadList(plansData, "data").filter(plan => isVpsPlanOrderable(plan));
  if (availablePlans.length === 0) {
    console.log("No VPS plan is currently available for this template.");
    printOutput(plansData, false);
    return;
  }

  const plan = await select(rl, "Select VPS plan", availablePlans.map(plan => ({
    label: `${plan.name} (${plan.cpu} CPU, ${plan.ram}GB RAM, ${plan.disk}GB disk)`,
    hint: getVpsPlanChoiceHint(plan),
    value: plan
  })));
  if (!plan) return;

  const duration = await select(rl, "Select duration", pricedDurationChoices(plan));
  if (!duration) return;

  const name = await ask(rl, "VPS name", `drite-vps-${Date.now()}`);
  const password = await ask(rl, "Root password");
  if (!password) return;

  await runAndShow(rl, {
    method: "POST",
    path: "/vps",
    body: {
      name,
      templateId: template.id,
      planId: plan.id,
      durationType: duration,
      password
    }
  });
}

async function interactiveVpsManage(rl: ReturnType<typeof createInterface>) {
  const vps = await selectVps(rl);
  if (!vps) return;

  while (true) {
    const action = await select(rl, `VPS: ${vps.name || vps.id}`, [
      { label: "Detail", value: "get" },
      { label: "Live status", value: "status" },
      { label: "Stats", value: "stats" },
      { label: "Activity", value: "activity" },
      { label: "Start", value: "start" },
      { label: "Stop", value: "stop" },
      { label: "Reboot", value: "reboot" },
      { label: "Force stop", value: "force-stop" },
      { label: "Rename", value: "rename" },
      { label: "Renew", value: "renew" },
      { label: "Upgrade", value: "upgrade" },
      { label: "Set auto renew", value: "auto-renew" },
      { label: "Reset password", value: "reset-password" },
      { label: "Delete", value: "delete" }
    ]);
    if (!action) return;

    if (["get", "status", "stats", "activity"].includes(action)) {
      const suffix = action === "get" ? "" : `/${action}`;
      await runAndShow(rl, { path: `/vps/${vps.id}${suffix}` });
      continue;
    }

    if (["start", "stop", "reboot", "force-stop"].includes(action)) {
      if (await confirm(rl, `Run ${action} on ${vps.name || vps.id}?`)) {
        await runAndShow(rl, { method: "POST", path: `/vps/${vps.id}/${action}` });
      }
      continue;
    }

    if (action === "rename") {
      const name = await ask(rl, "New VPS name");
      if (name.trim().length > 0) {
        await runAndShow(rl, { method: "POST", path: `/vps/${vps.id}/rename`, body: { name: name.trim() } });
        vps.name = name.trim();
      }
      continue;
    }

    if (action === "renew") {
      const duration = await select(rl, "Select renew duration", pricedDurationChoices(vps.plan));
      if (duration && await confirm(rl, `Renew ${vps.name || vps.id} for ${duration}?`)) {
        await runAndShow(rl, { method: "POST", path: `/vps/${vps.id}/renew`, body: { durationType: duration } });
      }
      continue;
    }

    if (action === "upgrade") {
      const optionsData = await fetchInteractive({ path: `/vps/${vps.id}/upgrade-options` });
      const options = asArray((optionsData as any)?.data?.options).filter(option => option?.capacity?.canUpgrade);
      const selected = await select(rl, "Select upgrade plan", options.map(option => ({
        label: `${option.plan?.name || option.plan?.id} (${option.newSpec?.cpu} CPU, ${option.newSpec?.ram}GB RAM, ${option.newSpec?.disk}GB disk)`,
        hint: `${option.charge || 0} THB now, next ${option.nextPrice || 0} THB`,
        value: option
      })));
      if (selected && await confirm(rl, `Upgrade ${vps.name || vps.id} to ${selected.plan?.name || selected.plan?.id}?`)) {
        await runAndShow(rl, { method: "POST", path: `/vps/${vps.id}/upgrade`, body: { planId: selected.plan.id } });
      }
      continue;
    }

    if (action === "auto-renew") {
      const enabled = await select(rl, "Auto renew", [
        { label: "Enable", value: true },
        { label: "Disable", value: false }
      ]);
      if (enabled !== null) {
        await runAndShow(rl, { method: "POST", path: `/vps/${vps.id}/auto-renewal`, body: { enabled } });
      }
      continue;
    }

    if (action === "reset-password") {
      const password = await ask(rl, "New root password");
      if (password && await confirm(rl, `Reset password for ${vps.name || vps.id}?`)) {
        await runAndShow(rl, { method: "POST", path: `/vps/${vps.id}/reset-password`, body: { password } });
      }
      continue;
    }

    if (action === "delete" && await confirm(rl, `Delete ${vps.name || vps.id}?`)) {
      await runAndShow(rl, { method: "DELETE", path: `/vps/${vps.id}` });
    }
  }
}

async function interactiveVps(rl: ReturnType<typeof createInterface>) {
  while (true) {
    const action = await select(rl, "VPS", [
      { label: "List VPS", value: "list" },
      { label: "Create VPS", value: "create" },
      { label: "Manage VPS", value: "manage" },
      { label: "Plans", value: "plans" },
      { label: "Templates", value: "templates" },
      { label: "Failed jobs", value: "failed" }
    ]);
    if (!action) return;

    if (action === "create") await interactiveVpsCreate(rl);
    else if (action === "manage") await interactiveVpsManage(rl);
    else await runAndShow(rl, { path: action === "list" ? "/vps" : `/vps/${action}` });
  }
}

async function interactiveHostingDeploy(rl: ReturnType<typeof createInterface>) {
  const plansData = await fetchInteractive({ path: "/api/un_auth/hosting/plans" });
  if (!plansData) return;

  const plan = await select(rl, "Select hosting plan", getPayloadList(plansData, "plans").map(plan => ({
    label: `${plan.name} (${plan.disk}MB disk, ${plan.traffic}MB traffic)`,
    hint: `daily ${plan.dailyPrice}, monthly ${plan.monthlyPrice}`,
    value: plan
  })));
  if (!plan) return;

  const duration = await select(rl, "Select duration", pricedHostingDurationChoices(plan));
  if (!duration) return;

  const domain = await ask(rl, "Domain");
  const password = await ask(rl, "Plesk password");
  if (!domain || !password) return;

  await runAndShow(rl, {
    method: "POST",
    path: "/hosting/deploy",
    body: { duration, domain, planId: plan.id, password }
  });
}

async function interactiveHostingManage(rl: ReturnType<typeof createInterface>) {
  const hosting = await selectHosting(rl);
  if (!hosting) return;

  while (true) {
    const action = await select(rl, `Hosting: ${hosting.name || hosting.id}`, [
      { label: "Detail", value: "get" },
      { label: "Stats", value: "stats" },
      { label: "Disk", value: "disk" },
      { label: "Traffic", value: "traffic" },
      { label: "Activity", value: "activity" },
      { label: "Activation status", value: "activation-status" },
      { label: "Access link", value: "access" },
      { label: "Renew", value: "renew" },
      { label: "Toggle auto renew", value: "autorenew" },
      { label: "Reset password", value: "reset-password" },
      { label: "Delete", value: "delete" }
    ]);
    if (!action) return;

    if (["get", "stats", "disk", "traffic", "activity", "activation-status"].includes(action)) {
      const suffix = action === "get" ? "" : `/${action}`;
      await runAndShow(rl, { path: `/hosting/${hosting.id}${suffix}` });
      continue;
    }

    if (action === "access" || action === "autorenew") {
      await runAndShow(rl, { method: "POST", path: `/hosting/${hosting.id}/${action}` });
      continue;
    }

    if (action === "renew") {
      const duration = await select(rl, "Select renew duration", pricedDurationChoices(hosting.plan));
      if (duration && await confirm(rl, `Renew ${hosting.name || hosting.id} for ${duration}?`)) {
        await runAndShow(rl, { method: "POST", path: `/hosting/${hosting.id}/renew`, body: { durationType: duration } });
      }
      continue;
    }

    if (action === "reset-password") {
      const password = await ask(rl, "New Plesk password");
      if (password && await confirm(rl, `Reset password for ${hosting.name || hosting.id}?`)) {
        await runAndShow(rl, { method: "POST", path: `/hosting/${hosting.id}/reset-password`, body: { password } });
      }
      continue;
    }

    if (action === "delete" && await confirm(rl, `Delete hosting ${hosting.name || hosting.id}?`)) {
      await runAndShow(rl, { method: "DELETE", path: `/hosting/${hosting.id}` });
    }
  }
}

async function interactiveHosting(rl: ReturnType<typeof createInterface>) {
  while (true) {
    const action = await select(rl, "Hosting", [
      { label: "Plans", value: "plans" },
      { label: "List hosting", value: "list" },
      { label: "Deploy hosting", value: "deploy" },
      { label: "Manage hosting", value: "manage" },
      { label: "Check domain", value: "check-domain" }
    ]);
    if (!action) return;

    if (action === "plans") await runAndShow(rl, { path: "/api/un_auth/hosting/plans" });
    else if (action === "list") await runAndShow(rl, { path: "/hosting/list" });
    else if (action === "deploy") await interactiveHostingDeploy(rl);
    else if (action === "manage") await interactiveHostingManage(rl);
    else {
      const domain = await ask(rl, "Domain");
      if (domain) await runAndShow(rl, { path: "/hosting/check-domain", query: { domain } });
    }
  }
}

async function interactiveBilling(rl: ReturnType<typeof createInterface>) {
  const action = await select(rl, "Billing", [
    { label: "Transactions", value: "transactions" },
    { label: "Export transactions", value: "export" },
    { label: "Topup history", value: "topup-history" },
    { label: "Topup status", value: "topup-status" }
  ]);
  if (!action) return;

  if (action === "transactions") {
    await runAndShow(rl, { path: "/transactions", query: { page: "1", limit: "20" } });
  } else if (action === "export") {
    const month = await ask(rl, "Month YYYY-MM (blank for all)");
    await runAndShow(rl, { path: "/transactions/export", query: month ? { month } : {} });
  } else if (action === "topup-history") {
    await runAndShow(rl, { path: "/topup/history" });
  } else {
    const referenceNo = await ask(rl, "Reference number");
    if (referenceNo) await runAndShow(rl, { path: `/topup/status/${referenceNo}` });
  }
}

async function selectTicket(rl: ReturnType<typeof createInterface>) {
  const data = await fetchInteractive({ path: "/ticket/list", query: { page: "1", limit: "20", status: "all" } });
  const tickets = getPayloadList(data, "tickets");
  return select(rl, "Select ticket", tickets.map(ticket => ({
    label: ticket.subject || ticket.id,
    hint: `${ticket.status} / ${ticket.priority} / ${ticket.id}`,
    value: ticket
  })));
}

async function interactiveTicket(rl: ReturnType<typeof createInterface>) {
  const action = await select(rl, "Tickets", [
    { label: "List tickets", value: "list" },
    { label: "Create ticket", value: "create" },
    { label: "View ticket", value: "get" },
    { label: "Reply ticket", value: "reply" },
    { label: "Close ticket", value: "close" }
  ]);
  if (!action) return;

  if (action === "list") return runAndShow(rl, { path: "/ticket/list", query: { page: "1", limit: "20", status: "all" } });
  if (action === "create") {
    const subject = await ask(rl, "Subject");
    const category = await select(rl, "Category", ["technical", "billing", "sales", "security", "migration"].map(value => ({ label: value, value })));
    const priority = await select(rl, "Priority", ["low", "normal", "urgent"].map(value => ({ label: value, value })));
    const message = await ask(rl, "Message");
    if (subject && category && priority && message) {
      return runAndShow(rl, { method: "POST", path: "/ticket", body: { subject, category, priority, message } });
    }
    return;
  }

  const ticket = await selectTicket(rl);
  if (!ticket) return;
  if (action === "get") return runAndShow(rl, { path: `/ticket/${ticket.id}` });
  if (action === "reply") {
    const message = await ask(rl, "Reply");
    if (message) return runAndShow(rl, { method: "POST", path: `/ticket/${ticket.id}/reply`, body: { message } });
  }
  if (action === "close" && await confirm(rl, `Close ticket ${ticket.id}?`)) {
    return runAndShow(rl, { method: "PUT", path: `/ticket/${ticket.id}/close` });
  }
}

async function interactiveWebhooks(rl: ReturnType<typeof createInterface>) {
  const action = await select(rl, "Webhooks", [
    { label: "List webhooks", value: "list" },
    { label: "Create webhook", value: "create" },
    { label: "Delete webhook", value: "delete" }
  ]);
  if (!action) return;

  if (action === "list") return runAndShow(rl, { path: "/me/webhooks" });
  if (action === "create") {
    const url = await ask(rl, "Webhook URL");
    const events = (await ask(rl, "Events comma-separated", "vps.created")).split(",").map(event => event.trim()).filter(Boolean);
    if (url) return runAndShow(rl, { method: "POST", path: "/me/webhooks", body: { url, events } });
  }
  if (action === "delete") {
    const data = await fetchInteractive({ path: "/me/webhooks" });
    const webhook = await select(rl, "Select webhook", getPayloadList(data, "data").map(webhook => ({
      label: webhook.url,
      hint: `${webhook.status} / ${webhook.id}`,
      value: webhook
    })));
    if (webhook && await confirm(rl, `Delete webhook ${webhook.id}?`)) {
      return runAndShow(rl, { method: "DELETE", path: `/me/webhooks/${webhook.id}` });
    }
  }
}

async function interactiveApiKey(rl: ReturnType<typeof createInterface>) {
  const action = await select(rl, "API Key", [
    { label: "API logs", value: "logs" },
    { label: "Set allowed IPs", value: "security" },
    { label: "Generate new API key", value: "create" },
    { label: "Revoke API key", value: "revoke" }
  ]);
  if (!action) return;

  if (action === "logs") return runAndShow(rl, { path: "/me/api-logs", query: { page: "1", limit: "20" } });
  if (action === "security") {
    const raw = await ask(rl, "Allowed IPs comma-separated (blank allows all)");
    const allowedIps = raw.split(",").map(ip => ip.trim()).filter(Boolean);
    return runAndShow(rl, { method: "PUT", path: "/me/api-key/security", body: { allowedIps } });
  }
  if (action === "create" && await confirm(rl, "Generate a new API key? Existing key will be replaced.")) {
    return runAndShow(rl, { method: "POST", path: "/me/api-key" });
  }
  if (action === "revoke" && await confirm(rl, "Revoke current API key?")) {
    return runAndShow(rl, { method: "DELETE", path: "/me/api-key" });
  }
}

async function interactiveRaw(rl: ReturnType<typeof createInterface>) {
  const method = await select(rl, "HTTP method", [...METHODS].map(value => ({ label: value, value })));
  if (!method) return;
  const path = await ask(rl, "Path", "/api/auth/me");
  const bodyText = ["POST", "PUT", "PATCH"].includes(method) ? await ask(rl, "JSON body (blank for none)") : "";
  let body: unknown;
  if (bodyText) {
    try {
      body = JSON.parse(bodyText);
    } catch (error: any) {
      console.log(`Invalid JSON: ${error.message}`);
      await pause(rl);
      return;
    }
  }
  await runAndShow(rl, { method, path, body });
}

async function runInteractive() {
  await withReadline(async (rl) => {
    await ensureInteractiveAuth(rl);

    while (true) {
      const action = await select(rl, "Drite CLI", [
        { label: "Account profile", value: "me" },
        { label: "VPS", value: "vps" },
        { label: "Hosting", value: "hosting" },
        { label: "Billing", value: "billing" },
        { label: "Tickets", value: "ticket" },
        { label: "Webhooks", value: "webhook" },
        { label: "API key/security", value: "api-key" },
        { label: "Raw request", value: "raw" },
        { label: "Exit", value: "exit" }
      ]);

      if (!action || action === "exit") return;
      if (action === "me") await runAndShow(rl, { path: "/me" });
      else if (action === "vps") await interactiveVps(rl);
      else if (action === "hosting") await interactiveHosting(rl);
      else if (action === "billing") await interactiveBilling(rl);
      else if (action === "ticket") await interactiveTicket(rl);
      else if (action === "webhook") await interactiveWebhooks(rl);
      else if (action === "api-key") await interactiveApiKey(rl);
      else if (action === "raw") await interactiveRaw(rl);
    }
  });
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const [command, action] = args.positionals;

  if (!command || command === "interactive" || command === "menu") {
    await runInteractive();
    return;
  }

  if (command === "help" || command === "--help" || command === "-h") {
    printHelp();
    return;
  }

  switch (command) {
    case "auth":
      return handleAuth(args, action);
    case "config":
      return handleConfig(args, action);
    case "me":
      return apiRequest(args, { path: "/me" });
    case "doctor":
      return handleDoctor(args);
    case "api-logs":
      return apiRequest(args, { path: "/me/api-logs", query: parseQuery(args) });
    case "vps":
      return handleVps(args, action);
    case "hosting":
      return handleHosting(args, action);
    case "billing":
      return handleBilling(args, action);
    case "plans":
      return apiRequest(args, { path: action === "hosting" ? "/api/un_auth/hosting/plans" : "/api/un_auth/plans/all" });
    case "ticket":
      return handleTicket(args, action);
    case "webhook":
    case "webhooks":
      return handleWebhook(args, action);
    case "api-key":
      return handleApiKey(args, action);
    case "raw":
      return handleRaw(args);
    default:
      throw new CliError(`Unknown command: ${command}`);
  }
}

main().catch((error: any) => {
  if (error instanceof CliError) {
    console.error(error.message);
    process.exit(error.code);
  }

  console.error(error?.stack || error?.message || String(error));
  process.exit(1);
});
