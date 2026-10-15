import React from 'react';

interface Props {
  children: React.ReactNode;
  onRecover: () => void;
}

export default class AppErrorBoundary extends React.Component<Props, { failed: boolean }> {
  state = { failed: false };

  static getDerivedStateFromError() {
    return { failed: true };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error('Smallworld could not render the page', error, info.componentStack);
  }

  render() {
    if (!this.state.failed) return this.props.children;
    return (
      <main className="min-h-screen flex items-center justify-center bg-[#F5F5DC] p-6 font-serif text-[#5F4B32]">
        <div role="alert" className="max-w-md rounded-xl border border-[#8B4513] bg-[#FDF5E6] p-8 shadow-lg">
          <h1 className="mb-3 text-2xl font-bold">Let’s reopen Smallworld</h1>
          <p className="mb-6">The page couldn’t be displayed. Reopen it to reconnect to your game.</p>
          <button type="button" onClick={this.props.onRecover}
            className="rounded bg-[#8B4513] px-5 py-3 text-white hover:bg-[#A0522D] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-[#8B4513]">
            Reopen Smallworld
          </button>
        </div>
      </main>
    );
  }
}
