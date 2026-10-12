import { useQuery } from "@tanstack/react-query";
import { Navigate, Outlet } from "react-router";
import { Skeleton } from "@/components/ui/skeleton";
import { EngagementService } from "@/lib/api";
import { keys } from "@/lib/query";

const NO_ENGAGEMENT = "no engagement open";

export function RequireEngagement() {
  const q = useQuery({ queryKey: keys.engagement(), queryFn: () => EngagementService.Current() });
  if (q.isError && q.error.message.includes(NO_ENGAGEMENT)) return <Navigate to="/" replace />;
  if (q.isPending) return <Skeleton className="m-6 h-40" />;
  return <Outlet />;
}
