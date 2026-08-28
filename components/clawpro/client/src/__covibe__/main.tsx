import { createRoot } from "react-dom/client";

import App from "../App";
import ErrorBoundary from "../components/ErrorBoundary";
import { Toaster } from "../components/ui/sonner";
import { TooltipProvider } from "../components/ui/tooltip";
import { ThemeProvider } from "../contexts/ThemeContext";
import { UserRoleProvider } from "../contexts/UserRoleContext";
import DesignSystemComponents from "../pages/DesignSystemComponents";
import "../index.css";

function ShowcaseRoot() {
  const path = window.location.pathname;
  const isShowcaseHome = path === "/" || path === "/design-system/components";

  if (!isShowcaseHome) {
    return <App />;
  }

  return (
    <ErrorBoundary>
      <ThemeProvider defaultTheme="light">
        <UserRoleProvider>
          <TooltipProvider>
            <DesignSystemComponents />
            <Toaster position="top-right" closeButton />
          </TooltipProvider>
        </UserRoleProvider>
      </ThemeProvider>
    </ErrorBoundary>
  );
}

createRoot(document.getElementById("root")!).render(<ShowcaseRoot />);
