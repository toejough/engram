import { BrowserRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import Dashboard from "./pages/Dashboard";
import MemoryDetail from "./pages/MemoryDetail";
import Surface from "./pages/Surface";
import Projects from "./pages/Projects";
import Activity from "./pages/Activity";

const queryClient = new QueryClient();

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/memories/:slug" element={<MemoryDetail />} />
          <Route path="/surface" element={<Surface />} />
          <Route path="/projects" element={<Projects />} />
          <Route path="/activity" element={<Activity />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
