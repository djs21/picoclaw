import type { TFunction } from "i18next"

import type { ToolSupportItem } from "@/api/tools"

type MarketplaceTool =
  | Pick<ToolSupportItem, "status" | "reason_code">
  | undefined

export interface UnavailableToolMessage {
  key: "skills"
  label: string
  message: string
}

// buildUnavailableToolMessages explains why the marketplace is unavailable.
// Skill registry access is gated by the `skills` subsystem alone now that the
// find_skills and install_skill agent tools were removed, so at most one
// message is returned.
export function buildUnavailableToolMessages({
  skillsTool,
  t,
}: {
  skillsTool: MarketplaceTool
  t: TFunction
}): UnavailableToolMessage[] {
  const message = getToolSupportMessage(skillsTool, t)
  if (!message) {
    return []
  }
  return [
    {
      key: "skills",
      label: t("pages.agent.skills.marketplace_search_status"),
      message,
    },
  ]
}

function getToolSupportMessage(
  tool: MarketplaceTool,
  t: TFunction,
): string | null {
  if (!tool || tool.status === "enabled") {
    return null
  }
  if (tool.reason_code) {
    return `${t(`pages.agent.tools.reasons.${tool.reason_code}`)} ${t("pages.agent.skills.marketplace_status_enable_hint")}`
  }
  return t("pages.agent.skills.marketplace_status_disabled")
}
