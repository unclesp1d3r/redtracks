import { QueryClientProvider } from "@tanstack/react-query";
import { createHashRouter, RouterProvider } from "react-router";
import { AppShell } from "@/components/AppShell";
import { RequireEngagement } from "@/components/RequireEngagement";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { BoardPage } from "@/features/board/BoardPage";
import { EngagementsPage } from "@/features/engagements/EngagementsPage";
import { EvidencePage } from "@/features/evidence/EvidencePage";
import { HostPage } from "@/features/host/HostPage";
import { ImportPage } from "@/features/import/ImportPage";
import { InboxPanel } from "@/features/inbox/InboxPanel";
import { WhatsLeftPage } from "@/features/left/WhatsLeftPage";
import { SessionLogsPage } from "@/features/sessionlogs/SessionLogsPage";
import { TablesPage } from "@/features/tables/TablesPage";
import { queryClient } from "@/lib/query";

export const routes = [
  { path: "/", element: <EngagementsPage /> },
  {
    element: <RequireEngagement />,
    children: [
      {
        element: <AppShell />,
        children: [
          { path: "/board", element: <BoardPage /> },
          { path: "/host/:ip", element: <HostPage /> },
          { path: "/tables", element: <TablesPage /> },
          { path: "/import", element: <ImportPage /> },
          { path: "/left", element: <WhatsLeftPage /> },
          { path: "/evidence", element: <EvidencePage /> },
          { path: "/inbox", element: <InboxPanel /> },
          { path: "/logs", element: <SessionLogsPage /> },
        ],
      },
    ],
  },
];

const router = createHashRouter(routes);

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <RouterProvider router={router} />
        <Toaster />
      </TooltipProvider>
    </QueryClientProvider>
  );
}

export default App;
