import { Component } from "react";
import type { ReactNode, ErrorInfo } from "react";
import ErrorState from "./ErrorState";

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export default class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("ErrorBoundary caught:", error, info.componentStack);
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError) {
      return (
        <div className="mx-auto max-w-7xl p-8">
          <ErrorState
            message={
              this.state.error?.message ?? "An unexpected error occurred."
            }
            onRetry={this.handleRetry}
          />
        </div>
      );
    }

    return this.props.children;
  }
}
