import { Outlet } from 'react-router-dom';
import Header from './Header';
import Footer from './Footer';
import { ToastProvider } from '@/components/ui/Toaster';

export default function Layout() {
  return (
    <ToastProvider>
      <div className="flex min-h-screen flex-col bg-white dark:bg-asgard-950">
        <Header />
        <main className="flex-1">
          <Outlet />
        </main>
        <Footer />
      </div>
    </ToastProvider>
  );
}
