import { decorateVpsPlansPayload } from "../vps/availability";

type ApiResult = {
  ok: boolean;
  status: number;
  data: unknown;
};

type RequestOptions = {
  path: string;
  query?: Record<string, string>;
};

type VpsPlansCommandOptions<TArgs> = {
  args: TArgs;
  compact: boolean;
  query: Record<string, string>;
  templateId?: string;
  availableOnly?: boolean;
  requestJson: (args: TArgs, options: RequestOptions) => Promise<ApiResult>;
  printOutput: (data: unknown, compact: boolean) => void;
};

function filterUnavailable(payload: unknown) {
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) return payload;
  const root = payload as Record<string, any>;
  const plans = Array.isArray(root.data)
    ? root.data
    : Array.isArray(root.plans)
      ? root.plans
      : null;

  if (!plans) return payload;

  const filtered = plans.filter((plan: any) => plan?.isOrderable !== false);
  return {
    ...root,
    ...(Array.isArray(root.data) ? { data: filtered } : { plans: filtered })
  };
}

export async function handleVpsPlansCommand<TArgs>({
  args,
  compact,
  query,
  templateId,
  availableOnly,
  requestJson,
  printOutput
}: VpsPlansCommandOptions<TArgs>) {
  const finalQuery = { ...query };
  if (templateId) finalQuery.templateId = templateId;

  const result = await requestJson(args, {
    path: "/vps/plans",
    query: finalQuery
  });

  if (!result.ok) {
    printOutput({ ok: false, status: result.status, data: result.data }, compact);
    process.exit(result.status >= 400 && result.status < 600 ? 1 : 0);
  }

  const decorated = decorateVpsPlansPayload(result.data);
  printOutput(availableOnly ? filterUnavailable(decorated) : decorated, compact);
}
