import { useEffect } from "react";
import {
  BrowserRouter,
  Routes,
  Route,
  useLocation,
} from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "@/components/ui/sonner";
import ErrorBoundary from "@/components/ErrorBoundary";
import Dashboard from "./pages/Dashboard";
import MemoryDetail from "./pages/MemoryDetail";
import Surface from "./pages/Surface";
import Projects from "./pages/Projects";
import Activity from "./pages/Activity";

const queryClient = new QueryClient();

function FocusManager() {
  const location = useLocation();

  useEffect(() => {
    // Reset focus to top of page on route change for keyboard navigation
    const main = document.querySelector<HTMLElement>("h1");
    if (main) {
      main.tabIndex = -1;
      main.focus({ preventScroll: false });
      main.removeAttribute("tabindex");
    }
  }, [location.pathname]);

  return null;
}

function EscapeHandler() {
  useEffect(() => {
    function handleEscape(e: KeyboardEvent) {
      if (e.key !== "Escape") return;

      // Close any open dialogs — find the topmost open base-ui dialog backdrop
      const openBackdrop = document.querySelector("[data-open]");
      if (openBackdrop) {
        // base-ui AlertDialog already handles Escape natively, but this is a safety net
        return;
      }
    }

    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, []);

  return null;
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <ErrorBoundary>
          <FocusManager />
          <EscapeHandler />
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/memories/:slug" element={<MemoryDetail />} />
            <Route path="/surface" element={<Surface />} />
            <Route path="/projects" element={<Projects />} />
            <Route path="/activity" element={<Activity />} />
          </Routes>
        </ErrorBoundary>
      </BrowserRouter>
      <Toaster />
    </QueryClientProvider>
  );
}
