import type { ReactNode } from "react";
import type { UseQueryResult } from "@tanstack/react-query";
import { TriangleAlertIcon } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

interface Props<T> {
  query: UseQueryResult<T>;
  /** Empty state including the page's primary action. */
  empty: ReactNode;
  isEmpty: (data: T) => boolean;
  children: (data: T) => ReactNode;
}

/** Shared pending / empty / error rendering for every page. */
export function QueryStates<T>({ query, empty, isEmpty, children }: Props<T>) {
  if (query.isPending) {
    return (
      <div className="space-y-2" data-testid="query-pending">
        <Skeleton className="h-8 w-1/3" />
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-24 w-full" />
      </div>
    );
  }
  if (query.isError) {
    return (
      <Alert variant="destructive">
        <TriangleAlertIcon />
        <AlertTitle>Could not load</AlertTitle>
        <AlertDescription>
          <p className="font-mono text-xs whitespace-pre-wrap">{query.error.message}</p>
          <Button className="mt-2" variant="outline" size="sm" onClick={() => void query.refetch()}>
            Retry
          </Button>
        </AlertDescription>
      </Alert>
    );
  }
  if (isEmpty(query.data)) return <>{empty}</>;
  return <>{children(query.data)}</>;
}
