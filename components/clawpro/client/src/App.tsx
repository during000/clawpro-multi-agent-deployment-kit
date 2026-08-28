import { useState, lazy, Suspense } from "react";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { Route, Switch } from "wouter";
import ErrorBoundary from "./components/ErrorBoundary";
import { ThemeProvider } from "./contexts/ThemeContext";
import { UserRoleProvider } from "./contexts/UserRoleContext";
import { PluginUpgradeProvider } from "./contexts/PluginUpgradeContext";
import { PluginUpgradeFloating } from "./components/PluginUpgradeFloating";
import { OnboardingDemoPanel } from "./components/onboarding";
import AdminLayout from "./components/AdminLayout";
import ModeAwareRoute from "./components/ModeAwareRoute";

const NotFound = lazy(() => import("@/pages/NotFound"));

// Demo
const SsoLoginDemo = lazy(() => import("./pages/SsoLoginDemo"));
const ComponentPreview = lazy(() => import("./pages/ComponentPreview"));
const FilterPanelPreview = lazy(() => import("./pages/admin/ComponentPreview"));
const DesignSystemComponents = lazy(() => import("./pages/DesignSystemComponents"));
const DesignSystemAssets = lazy(() => import("./pages/DesignSystemAssets"));

// Landing
const LandingPageV2 = lazy(() => import("./pages/landing"));
const PreviewIndex = lazy(() => import("./pages/PreviewIndex"));
const EmptyStatePreview = lazy(() => import("./pages/preview/EmptyStatePreview"));
const DataTablePreview = lazy(() => import("./pages/preview/DataTablePreview"));
const EmptyStateSpecVerify = lazy(() => import("./pages/preview/EmptyStateSpecVerify"));
const TablePreview = lazy(() => import("./pages/preview/TablePreview"));
const ToastPreview = lazy(() => import("./pages/preview/ToastPreview"));
const AvatarPreview = lazy(() => import("./pages/preview/AvatarPreview"));
const TreePreview = lazy(() => import("./pages/preview/TreePreview"));
const BreadcrumbPreview = lazy(() => import("./pages/preview/BreadcrumbPreview"));
const TransferPreview = lazy(() => import("./pages/preview/TransferPreview"));
const AlertPreview = lazy(() => import("./pages/preview/AlertPreview"));
const ButtonPreview = lazy(() => import("./pages/preview/ButtonPreview"));
const DatePickerPreview = lazy(() => import("./pages/preview/DatePickerPreview"));
const SkillMapAbDemo = lazy(() => import("./pages/preview/SkillMapAbDemo"));
const DateTimePickerPreview = lazy(() => import("./pages/preview/DateTimePickerPreview"));
const OnboardingGuidePreview = lazy(() => import("./pages/preview/OnboardingGuidePreview"));

// Tenant
const MyAgent = lazy(() => import("./pages/tenant/MyOpenClaw"));
const AgentDetail = lazy(() => import("./pages/tenant/OpenClawDetail"));
const OpenClawDetailGuide = lazy(() => import("./pages/tenant/OpenClawDetailGuide"));
const ModelQuota = lazy(() => import("./pages/tenant/ModelQuota"));
const HelpDocs = lazy(() => import("./pages/tenant/HelpDocs"));
const SkillSquare = lazy(() => import("./pages/tenant/SkillSquare"));
const TenantIconAudit = lazy(() => import("./pages/tenant/TenantIconAudit"));
const ResourceManagement = lazy(() => import("./pages/admin/ResourceManagement"));
const AgentChat = lazy(() => import("./pages/tenant/AgentChat"));
const ProjectCollaboration = lazy(() => import("./pages/tenant/ProjectCollaboration"));
const TeamAssets = lazy(() => import("./pages/tenant/TeamAssets"));

// Admin
const BasicInfo = lazy(() => import("./pages/admin/BasicInfo"));
const PlatformPolicy = lazy(() => import("./pages/admin/PlatformPolicy"));
const MemberManagement = lazy(() => import("./pages/admin/MemberManagement"));
const ModelConfig = lazy(() => import("./pages/admin/ModelConfig"));
const ChannelConfig = lazy(() => import("./pages/admin/ChannelConfig"));
const SkillConfig = lazy(() => import("./pages/admin/SkillConfig"));
const ProjectAssetManagement = lazy(() => import("./pages/admin/project-assets/ProjectAssetManagement"));
const ImageManagement = lazy(() => import("./pages/admin/ImageManagement"));
const SecurityGroupManagement = lazy(() => import("./pages/admin/SecurityGroupManagement"));
const CloudDevManagement = lazy(() => import("./pages/admin/CloudDevManagement"));
const CloudDevActivation = lazy(() => import("./pages/admin/CloudDevActivation"));
const KnowledgeManagement = lazy(() => import("./pages/admin/KnowledgeManagement"));
const AgentMonitor = lazy(() => import("./pages/admin/OpenClawMonitor"));
const AgentMigration = lazy(() => import("./pages/admin/AgentMigration"));
const TokensMonitor = lazy(() => import("./pages/admin/TokensMonitor"));
const AuditLog = lazy(() => import("./pages/admin/AuditLog"));
const SecurityManagement = lazy(() => import("./pages/admin/Security/index"));
const CredentialManagement = lazy(() => import("./pages/admin/CredentialManagement"));
const SessionManagement = lazy(() => import("./pages/admin/SessionManagement"));
const SessionDetail = lazy(() => import("./pages/admin/SessionDetail"));
const OpsObservation = lazy(() => import("./pages/admin/OpsObservation"));
const MemoryManagement = lazy(() => import("./pages/admin/MemoryManagement"));
const FileManagement = lazy(() => import("./pages/admin/FileManagement"));
const SkillDetailPage = lazy(() => import("./pages/admin/SkillDetailPage"));
const AgentToolLibrary = lazy(() => import("./pages/admin/AgentToolLibrary"));
const ApiDocs = lazy(() => import("./pages/admin/ApiDocs"));
const AgentCommandsPage = lazy(() => import("./pages/admin/agentOps/AgentCommandsPage"));
const StandardBasicInfo = lazy(() => import("./pages/admin/standard/StandardBasicInfo"));

function CloudDevRoute() {
  const [activated, setActivated] = useState(false);
  if (!activated) {
    return <CloudDevActivation onActivated={() => setActivated(true)} />;
  }
  return <CloudDevManagement />;
}

function Router() {
  return (
    <Switch>
      {/* Landing Page */}
      <Route path="/" component={LandingPageV2} />

      {/* Preview - 全站页面索引，方便 demo */}
      <Route path="/preview" component={PreviewIndex} />
      <Route path="/preview/empty-state" component={EmptyStatePreview} />
      <Route path="/preview/empty-state-spec-verify" component={EmptyStateSpecVerify} />
      <Route path="/preview/table" component={TablePreview} />
      <Route path="/preview/data-table" component={DataTablePreview} />
      <Route path="/preview/toast" component={ToastPreview} />
      <Route path="/preview/avatar" component={AvatarPreview} />
      <Route path="/preview/tree" component={TreePreview} />
      <Route path="/preview/breadcrumb" component={BreadcrumbPreview} />
      <Route path="/preview/transfer" component={TransferPreview} />
      <Route path="/preview/alert" component={AlertPreview} />
      <Route path="/preview/button" component={ButtonPreview} />
      <Route path="/preview/date-picker" component={DatePickerPreview} />
      <Route path="/preview/skill-map-ab" component={SkillMapAbDemo} />
      <Route path="/preview/date-time-picker" component={DateTimePickerPreview} />
      <Route path="/preview/onboarding-guide" component={OnboardingGuidePreview} />

       {/* Demo */}
      <Route path="/demo/sso-login" component={SsoLoginDemo} />
      <Route path="/component-preview/:name" component={ComponentPreview} />
      <Route path="/design-system/components" component={DesignSystemComponents} />
      <Route path="/design-system/assets" component={DesignSystemAssets} />
      <Route path="/filter-panel-preview" component={FilterPanelPreview} />

      {/* Tenant Routes */}
      <Route path="/my-openclaw" component={MyAgent} />
      <Route path="/project-collaboration" component={ProjectCollaboration} />
      <Route path="/openclaw/:id" component={OpenClawDetailGuide} />
      <Route path="/openclaw-guide" component={OpenClawDetailGuide} />
      <Route path="/openclaw-guide/:id" component={OpenClawDetailGuide} />
      <Route path="/model-quota" component={ModelQuota} />
      <Route path="/skill-square" component={SkillSquare} />
      <Route path="/team-assets" component={TeamAssets} />
      <Route path="/agent-hub" component={TeamAssets} />
      <Route path="/agent-community" component={TeamAssets} />
      <Route path="/help-docs" component={HelpDocs} />
      <Route path="/tenant-icon-audit" component={TenantIconAudit} />

      {/* Preview - Figma 还原稿 */}
      <Route path="/preview/agent-chat" component={() => <AgentChat />} />

      {/* Admin Routes - 使用顶层路由避免 wouter 嵌套路由匹配问题 */}
      <Route path="/admin/basic-info" component={() => <AdminLayout><ModeAwareRoute standard={<StandardBasicInfo />} custom={<BasicInfo />} /></AdminLayout>} />
      <Route path="/admin/platform-policy" component={() => <AdminLayout><PlatformPolicy /></AdminLayout>} />
      <Route path="/admin/members" component={() => <AdminLayout><MemberManagement /></AdminLayout>} />
      <Route path="/admin/model-config" component={() => <AdminLayout><ModelConfig /></AdminLayout>} />
      <Route path="/admin/channel-config" component={() => <AdminLayout><ChannelConfig /></AdminLayout>} />
      <Route path="/admin/skill-config" component={() => <AdminLayout><SkillConfig /></AdminLayout>} />
      <Route path="/admin/agent-template" component={() => <AdminLayout><ResourceManagement /></AdminLayout>} />
      <Route path="/admin/agent-template-list" component={() => <AdminLayout><ProjectAssetManagement /></AdminLayout>} />
      <Route path="/admin/image-management" component={() => <AdminLayout><ImageManagement /></AdminLayout>} />
      <Route path="/admin/agent-types" component={() => <AdminLayout><ImageManagement /></AdminLayout>} />
      <Route path="/admin/security-group" component={() => <AdminLayout><SecurityGroupManagement /></AdminLayout>} />
      <Route path="/admin/knowledge-management" component={() => <AdminLayout><KnowledgeManagement /></AdminLayout>} />
      <Route path="/admin/cloud-dev" component={() => <AdminLayout><CloudDevRoute /></AdminLayout>} />
      <Route path="/admin/openclaw-monitor" component={() => <AdminLayout><AgentMonitor /></AdminLayout>} />
      <Route path="/admin/agent-commands" component={() => <AdminLayout><AgentCommandsPage /></AdminLayout>} />
      <Route path="/admin/agent-migration" component={() => <AdminLayout><AgentMigration /></AdminLayout>} />
      <Route path="/admin/tokens-monitor" component={() => <AdminLayout><TokensMonitor /></AdminLayout>} />
      <Route path="/admin/security-management" component={() => <SecurityManagement />} />
      <Route path="/admin/credential-management" component={() => <AdminLayout><CredentialManagement /></AdminLayout>} />
      <Route path="/admin/session/:id" component={({ params }) => <AdminLayout><SessionDetail params={params} /></AdminLayout>} />
      <Route path="/admin/session-management" component={() => <AdminLayout><SessionManagement /></AdminLayout>} />
      <Route path="/admin/ops-observation" component={() => <AdminLayout><OpsObservation /></AdminLayout>} />
      <Route path="/admin/audit-log" component={() => <AdminLayout><AuditLog /></AdminLayout>} />
      <Route path="/admin/memory-management" component={() => <AdminLayout><MemoryManagement /></AdminLayout>} />
      <Route path="/admin/file-management" component={() => <AdminLayout><FileManagement /></AdminLayout>} />
      <Route path="/admin/skill-detail/:id" component={({ params }) => <AdminLayout><SkillDetailPage skillId={params.id} /></AdminLayout>} />
      <Route path="/admin/agent-tool-library" component={() => <AdminLayout><AgentToolLibrary /></AdminLayout>} />
      <Route path="/admin/api-docs" component={() => <AdminLayout><ApiDocs /></AdminLayout>} />

      <Route path="/404" component={NotFound} />
      <Route component={NotFound} />
    </Switch>
  );
}

function App() {
  return (
    <ErrorBoundary>
      <ThemeProvider defaultTheme="light">
        <UserRoleProvider>
          <TooltipProvider>
            <PluginUpgradeProvider>
              <Suspense fallback={<div className="flex items-center justify-center h-screen text-muted-foreground">加载中...</div>}>
                <Router />
              </Suspense>
              <Toaster position="top-right" closeButton />
              {/* CLS 采集插件升级进度浮窗（右下角，跨页面常驻） */}
              <PluginUpgradeFloating />
              {/* 引导体系体验面板（右下角，跨页面常驻） */}
              <OnboardingDemoPanel />
            </PluginUpgradeProvider>
          </TooltipProvider>
        </UserRoleProvider>
      </ThemeProvider>
    </ErrorBoundary>
  );
}

export default App;
