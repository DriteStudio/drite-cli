export type VpsPlanLike = {
  id?: unknown;
  name?: unknown;
  cpu?: unknown;
  ram?: unknown;
  disk?: unknown;
  dailyPrice?: unknown;
  monthlyPrice?: unknown;
  isOrderable?: unknown;
  isPublic?: unknown;
  availabilityStatus?: unknown;
  availabilityReason?: unknown;
  nextIpAvailableAt?: unknown;
};

function isRecord(value: unknown): value is Record<string, any> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function stringValue(value: unknown, fallback = "") {
  return typeof value === "string" && value.length > 0 ? value : fallback;
}

function formatDate(value: unknown) {
  const raw = stringValue(value);
  if (!raw) return "";
  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleString("th-TH", {
    dateStyle: "medium",
    timeStyle: "short"
  });
}

export function isVpsPlanOrderable(plan: VpsPlanLike) {
  if (typeof plan.isOrderable === "boolean") return plan.isOrderable;
  if (typeof plan.isPublic === "boolean") return plan.isPublic;
  return true;
}

export function getVpsAvailabilityBadge(plan: VpsPlanLike) {
  const status = stringValue(plan.availabilityStatus);
  if (status === "out_of_stock") return "IP หมด";
  if (status === "no_capacity") return "เครื่องยังไม่พร้อม";
  if (!isVpsPlanOrderable(plan)) return "ยังเช่าไม่ได้";
  return "พร้อมเช่า";
}

export function getVpsAvailabilityMessage(plan: VpsPlanLike) {
  const status = stringValue(plan.availabilityStatus);

  if (status === "out_of_stock") {
    const availableAt = formatDate(plan.nextIpAvailableAt);
    if (availableAt) {
      return `IP สำหรับเช่า VPS หมดชั่วคราว คาดว่าจะว่างหลัง ${availableAt} เพราะต้องแปลง VM ที่หมดอายุเป็น template ก่อน`;
    }
    return "IP สำหรับเช่า VPS หมดชั่วคราว IP จะว่างหลังระบบแปลง VM ที่หมดอายุเป็น template แล้ว";
  }

  if (status === "no_capacity") {
    return "ขณะนี้สเปคนี้ยังไม่มีเครื่องที่พร้อมรองรับ กรุณาเลือกแพลนเล็กลงหรือลองใหม่ภายหลัง";
  }

  const reason = stringValue(plan.availabilityReason);
  if (reason) return reason;

  return isVpsPlanOrderable(plan) ? "พร้อมเช่า" : "ยังไม่พร้อมให้เช่า";
}

export function getVpsPlanChoiceHint(plan: VpsPlanLike) {
  const badge = getVpsAvailabilityBadge(plan);
  const monthlyPrice = plan.monthlyPrice ?? "-";
  const dailyPrice = plan.dailyPrice ?? "-";
  const base = `daily ${dailyPrice}, monthly ${monthlyPrice}`;
  return isVpsPlanOrderable(plan) ? `${base} / ${badge}` : `${badge} - ${getVpsAvailabilityMessage(plan)}`;
}

export function decorateVpsPlansPayload(payload: unknown) {
  if (!isRecord(payload)) return payload;

  const sourcePlans = Array.isArray(payload.data)
    ? payload.data
    : Array.isArray(payload.plans)
      ? payload.plans
      : null;

  if (!sourcePlans) return payload;

  const plans = sourcePlans.map((plan) => {
    if (!isRecord(plan)) return plan;
    return {
      ...plan,
      availabilityBadge: getVpsAvailabilityBadge(plan),
      availabilityMessage: getVpsAvailabilityMessage(plan)
    };
  });

  return {
    ...payload,
    ...(Array.isArray(payload.data) ? { data: plans } : { plans }),
    availabilitySummary: {
      total: plans.length,
      orderable: plans.filter((plan) => isRecord(plan) && isVpsPlanOrderable(plan)).length,
      blocked: plans.filter((plan) => isRecord(plan) && !isVpsPlanOrderable(plan)).length
    }
  };
}
