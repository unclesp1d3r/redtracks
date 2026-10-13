import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { CredentialsTable } from "./CredentialsTable";
import { FindingsTable } from "./FindingsTable";
import { LootTable } from "./LootTable";
import { SharesTable } from "./SharesTable";
import { UsersTable } from "./UsersTable";

export function TablesPage() {
  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold">Tables</h1>
      <Tabs defaultValue="credentials">
        <TabsList>
          <TabsTrigger value="credentials">Credentials</TabsTrigger>
          <TabsTrigger value="users">Users</TabsTrigger>
          <TabsTrigger value="shares">Shares</TabsTrigger>
          <TabsTrigger value="loot">Loot</TabsTrigger>
          <TabsTrigger value="findings">Findings</TabsTrigger>
        </TabsList>
        <TabsContent value="credentials"><CredentialsTable /></TabsContent>
        <TabsContent value="users"><UsersTable /></TabsContent>
        <TabsContent value="shares"><SharesTable /></TabsContent>
        <TabsContent value="loot"><LootTable /></TabsContent>
        <TabsContent value="findings"><FindingsTable /></TabsContent>
      </Tabs>
    </div>
  );
}
