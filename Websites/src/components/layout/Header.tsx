import { useState, useEffect, useRef } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { Menu, X, ChevronDown, Sun, Moon, User, LogOut, Plane, Crosshair, Shield, Monitor, Landmark, Users } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/Button';
import { useTheme } from '@/providers/ThemeProvider';
import { useAuth } from '@/providers/AuthProvider';

interface ProductLink {
  href: string;
  label: string;
  description: string;
  icon: React.ReactNode;
}

const productLinks: ProductLink[] = [
  // ASGARD Defense Systems
  {
    href: '/valkyrie',
    label: 'Valkyrie',
    description: 'Autonomous Flight',
    icon: <Plane className="w-5 h-5" />,
  },
  {
    href: '/pricilla',
    label: 'Pricilla',
    description: 'Precision Guidance',
    icon: <Crosshair className="w-5 h-5" />,
  },
  {
    href: '/giru',
    label: 'Giru',
    description: 'AI Security',
    icon: <Shield className="w-5 h-5" />,
  },
  // Aura Genesis Ecosystem
  {
    href: '/apex-os',
    label: 'APEX-OS-LQ',
    description: 'Autonomous Computing',
    icon: <Monitor className="w-5 h-5" />,
  },
  {
    href: '/foundation',
    label: 'Foundation',
    description: 'Community Governance',
    icon: <Landmark className="w-5 h-5" />,
  },
  {
    href: '/icf',
    label: 'ICF Program',
    description: 'Proof of Intelligence',
    icon: <Users className="w-5 h-5" />,
  },
];

export default function Header() {
  const [isScrolled, setIsScrolled] = useState(false);
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const [isUserMenuOpen, setIsUserMenuOpen] = useState(false);
  const [isProductsOpen, setIsProductsOpen] = useState(false);
  const [isMobileProductsOpen, setIsMobileProductsOpen] = useState(false);
  const productsRef = useRef<HTMLDivElement>(null);
  const { setTheme, resolvedTheme } = useTheme();
  const { user, isAuthenticated, signOut } = useAuth();
  const location = useLocation();

  useEffect(() => {
    const handleScroll = () => {
      setIsScrolled(window.scrollY > 10);
    };

    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  useEffect(() => {
    setIsMobileMenuOpen(false);
    setIsUserMenuOpen(false);
    setIsProductsOpen(false);
    setIsMobileProductsOpen(false);
  }, [location.pathname]);

  // Close products dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (productsRef.current && !productsRef.current.contains(event.target as Node)) {
        setIsProductsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const isActiveLink = (href: string) => {
    if (href === '/') {
      return location.pathname === '/';
    }
    return location.pathname.startsWith(href);
  };

  const isProductActive = productLinks.some((link) => location.pathname === link.href);

  return (
    <header
      className={cn(
        'fixed top-0 left-0 right-0 z-40 transition-all duration-300',
        isScrolled
          ? 'bg-white/80 dark:bg-asgard-950/80 backdrop-blur-xl border-b border-asgard-100 dark:border-asgard-800'
          : 'bg-transparent'
      )}
    >
      <div className="container-wide">
        <nav className="flex h-16 items-center justify-between">
          {/* Logo */}
          <Link to="/" className="flex items-center gap-3 group">
            <div className="relative w-9 h-9">
              <svg
                viewBox="0 0 36 36"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
                className="w-full h-full text-primary transition-transform group-hover:scale-105"
              >
                <circle cx="18" cy="18" r="16" stroke="currentColor" strokeWidth="2" />
                <path d="M18 6L26 22H10L18 6Z" fill="currentColor" />
                <circle cx="18" cy="28" r="3" fill="currentColor" />
              </svg>
            </div>
            <div className="flex flex-col">
              <span className="font-bold text-lg text-asgard-900 dark:text-white leading-tight">
                ASGARD
              </span>
              <span className="text-[10px] text-asgard-500 dark:text-asgard-400 leading-tight">
                by Arobi
              </span>
            </div>
          </Link>

          {/* Company Tagline - Desktop only */}
          <div className="hidden lg:flex items-center">
            <span className="text-xs text-asgard-500 dark:text-asgard-400 italic">
              Software, Robotics, Aerospace Intelligence & Defense Systems
            </span>
          </div>

          {/* Desktop Navigation */}
          <div className="hidden md:flex items-center gap-6">
            {/* Home Link */}
            <Link
              to="/"
              className={cn(
                'text-sm font-medium transition-colors',
                isActiveLink('/')
                  ? 'text-primary'
                  : 'text-asgard-600 hover:text-asgard-900 dark:text-asgard-400 dark:hover:text-white'
              )}
            >
              Home
            </Link>

            {/* Features Link */}
            <Link
              to="/features"
              className={cn(
                'text-sm font-medium transition-colors',
                isActiveLink('/features')
                  ? 'text-primary'
                  : 'text-asgard-600 hover:text-asgard-900 dark:text-asgard-400 dark:hover:text-white'
              )}
            >
              Features
            </Link>

            {/* Products Dropdown */}
            <div ref={productsRef} className="relative">
              <button
                onClick={() => setIsProductsOpen(!isProductsOpen)}
                className={cn(
                  'flex items-center gap-1 text-sm font-medium transition-colors',
                  isProductActive
                    ? 'text-primary'
                    : 'text-asgard-600 hover:text-asgard-900 dark:text-asgard-400 dark:hover:text-white'
                )}
              >
                Products
                <ChevronDown
                  className={cn(
                    'w-4 h-4 transition-transform',
                    isProductsOpen && 'rotate-180'
                  )}
                />
              </button>

              <AnimatePresence>
                {isProductsOpen && (
                  <motion.div
                    initial={{ opacity: 0, y: 10, scale: 0.95 }}
                    animate={{ opacity: 1, y: 0, scale: 1 }}
                    exit={{ opacity: 0, y: 10, scale: 0.95 }}
                    transition={{ duration: 0.15 }}
                    className="absolute left-0 mt-3 w-64 rounded-xl bg-white dark:bg-asgard-900 border border-asgard-100 dark:border-asgard-800 shadow-medium overflow-hidden"
                  >
                    <div className="p-2">
                      {productLinks.map((product) => (
                        <Link
                          key={product.href}
                          to={product.href}
                          className={cn(
                            'flex items-start gap-3 px-3 py-3 rounded-lg transition-colors',
                            location.pathname === product.href
                              ? 'bg-primary/5 text-primary'
                              : 'text-asgard-700 dark:text-asgard-300 hover:bg-asgard-50 dark:hover:bg-asgard-800'
                          )}
                        >
                          <div
                            className={cn(
                              'p-2 rounded-lg',
                              location.pathname === product.href
                                ? 'bg-primary/10 text-primary'
                                : 'bg-asgard-100 dark:bg-asgard-800 text-asgard-600 dark:text-asgard-400'
                            )}
                          >
                            {product.icon}
                          </div>
                          <div>
                            <div className="font-medium text-sm">{product.label}</div>
                            <div className="text-xs text-asgard-500 dark:text-asgard-400">
                              {product.description}
                            </div>
                          </div>
                        </Link>
                      ))}
                    </div>
                  </motion.div>
                )}
              </AnimatePresence>
            </div>

            {/* Pricing Link */}
            <Link
              to="/pricing"
              className={cn(
                'text-sm font-medium transition-colors',
                isActiveLink('/pricing')
                  ? 'text-primary'
                  : 'text-asgard-600 hover:text-asgard-900 dark:text-asgard-400 dark:hover:text-white'
              )}
            >
              Pricing
            </Link>

            {/* About Link */}
            <Link
              to="/about"
              className={cn(
                'text-sm font-medium transition-colors',
                isActiveLink('/about')
                  ? 'text-primary'
                  : 'text-asgard-600 hover:text-asgard-900 dark:text-asgard-400 dark:hover:text-white'
              )}
            >
              About
            </Link>

            {/* Contact Link */}
            <Link
              to="/contact"
              className={cn(
                'text-sm font-medium transition-colors',
                isActiveLink('/contact')
                  ? 'text-primary'
                  : 'text-asgard-600 hover:text-asgard-900 dark:text-asgard-400 dark:hover:text-white'
              )}
            >
              Contact
            </Link>
          </div>

          {/* Right Actions */}
          <div className="flex items-center gap-3">
            {/* Theme Toggle */}
            <button
              onClick={() => setTheme(resolvedTheme === 'dark' ? 'light' : 'dark')}
              className="p-2 rounded-xl text-asgard-500 hover:text-asgard-900 hover:bg-asgard-100 dark:hover:text-white dark:hover:bg-asgard-800 transition-colors"
              aria-label="Toggle theme"
            >
              {resolvedTheme === 'dark' ? (
                <Sun className="w-5 h-5" />
              ) : (
                <Moon className="w-5 h-5" />
              )}
            </button>

            {/* Auth Actions */}
            {isAuthenticated ? (
              <div className="relative">
                <button
                  type="button"
                  onClick={() => setIsUserMenuOpen(!isUserMenuOpen)}
                  className="flex items-center gap-2 p-2 rounded-xl hover:bg-asgard-100 dark:hover:bg-asgard-800 transition-colors"
                  aria-label="Toggle user menu"
                >
                  <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
                    <User className="w-4 h-4 text-primary" />
                  </div>
                  <ChevronDown className={cn(
                    'w-4 h-4 text-asgard-500 transition-transform',
                    isUserMenuOpen && 'rotate-180'
                  )} />
                </button>

                <AnimatePresence>
                  {isUserMenuOpen && (
                    <motion.div
                      initial={{ opacity: 0, y: 10, scale: 0.95 }}
                      animate={{ opacity: 1, y: 0, scale: 1 }}
                      exit={{ opacity: 0, y: 10, scale: 0.95 }}
                      transition={{ duration: 0.15 }}
                      className="absolute right-0 mt-2 w-56 rounded-xl bg-white dark:bg-asgard-900 border border-asgard-100 dark:border-asgard-800 shadow-medium overflow-hidden"
                    >
                      <div className="p-3 border-b border-asgard-100 dark:border-asgard-800">
                        <p className="font-medium text-sm text-asgard-900 dark:text-white truncate">
                          {user?.fullName}
                        </p>
                        <p className="text-xs text-asgard-500 truncate">
                          {user?.email}
                        </p>
                      </div>
                      <div className="p-1">
                        <Link
                          to="/dashboard"
                          className="flex items-center gap-3 px-3 py-2 text-sm text-asgard-700 dark:text-asgard-300 hover:bg-asgard-50 dark:hover:bg-asgard-800 rounded-lg transition-colors"
                        >
                          <User className="w-4 h-4" />
                          Dashboard
                        </Link>
                        <button
                          onClick={() => signOut()}
                          className="w-full flex items-center gap-3 px-3 py-2 text-sm text-danger hover:bg-danger/5 rounded-lg transition-colors"
                        >
                          <LogOut className="w-4 h-4" />
                          Sign Out
                        </button>
                      </div>
                    </motion.div>
                  )}
                </AnimatePresence>
              </div>
            ) : (
              <div className="hidden sm:flex items-center gap-2">
                <Link to="/signin">
                  <Button variant="ghost" size="sm">
                    Sign In
                  </Button>
                </Link>
                <Link to="/signup">
                  <Button size="sm">Get Started</Button>
                </Link>
              </div>
            )}

            {/* Mobile Menu Toggle */}
            <button
              onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
              className="md:hidden p-2 rounded-xl text-asgard-500 hover:text-asgard-900 hover:bg-asgard-100 dark:hover:text-white dark:hover:bg-asgard-800 transition-colors"
              aria-label="Toggle menu"
            >
              {isMobileMenuOpen ? (
                <X className="w-5 h-5" />
              ) : (
                <Menu className="w-5 h-5" />
              )}
            </button>
          </div>
        </nav>
      </div>

      {/* Mobile Menu */}
      <AnimatePresence>
        {isMobileMenuOpen && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            className="md:hidden bg-white dark:bg-asgard-950 border-b border-asgard-100 dark:border-asgard-800 overflow-hidden"
          >
            <div className="container-wide py-4 space-y-1">
              {/* Company Tagline - Mobile */}
              <div className="pb-3 mb-3 border-b border-asgard-100 dark:border-asgard-800">
                <span className="text-xs text-asgard-500 dark:text-asgard-400 italic">
                  Aerospace Intelligence & Defense Systems
                </span>
              </div>

              {/* Home Link */}
              <Link
                to="/"
                className={cn(
                  'block py-2 px-3 rounded-lg text-base font-medium transition-colors',
                  isActiveLink('/')
                    ? 'text-primary bg-primary/5'
                    : 'text-asgard-600 dark:text-asgard-400 hover:bg-asgard-50 dark:hover:bg-asgard-900'
                )}
              >
                Home
              </Link>

              {/* Features Link */}
              <Link
                to="/features"
                className={cn(
                  'block py-2 px-3 rounded-lg text-base font-medium transition-colors',
                  isActiveLink('/features')
                    ? 'text-primary bg-primary/5'
                    : 'text-asgard-600 dark:text-asgard-400 hover:bg-asgard-50 dark:hover:bg-asgard-900'
                )}
              >
                Features
              </Link>

              {/* Products Accordion */}
              <div>
                <button
                  onClick={() => setIsMobileProductsOpen(!isMobileProductsOpen)}
                  className={cn(
                    'w-full flex items-center justify-between py-2 px-3 rounded-lg text-base font-medium transition-colors',
                    isProductActive
                      ? 'text-primary bg-primary/5'
                      : 'text-asgard-600 dark:text-asgard-400 hover:bg-asgard-50 dark:hover:bg-asgard-900'
                  )}
                >
                  Products
                  <ChevronDown
                    className={cn(
                      'w-4 h-4 transition-transform',
                      isMobileProductsOpen && 'rotate-180'
                    )}
                  />
                </button>

                <AnimatePresence>
                  {isMobileProductsOpen && (
                    <motion.div
                      initial={{ opacity: 0, height: 0 }}
                      animate={{ opacity: 1, height: 'auto' }}
                      exit={{ opacity: 0, height: 0 }}
                      className="overflow-hidden"
                    >
                      <div className="pl-4 py-2 space-y-1">
                        {productLinks.map((product) => (
                          <Link
                            key={product.href}
                            to={product.href}
                            className={cn(
                              'flex items-center gap-3 py-2 px-3 rounded-lg transition-colors',
                              location.pathname === product.href
                                ? 'text-primary bg-primary/5'
                                : 'text-asgard-600 dark:text-asgard-400 hover:bg-asgard-50 dark:hover:bg-asgard-900'
                            )}
                          >
                            <div
                              className={cn(
                                'p-1.5 rounded-md',
                                location.pathname === product.href
                                  ? 'bg-primary/10 text-primary'
                                  : 'bg-asgard-100 dark:bg-asgard-800 text-asgard-500'
                              )}
                            >
                              {product.icon}
                            </div>
                            <div>
                              <div className="font-medium text-sm">{product.label}</div>
                              <div className="text-xs text-asgard-500">
                                {product.description}
                              </div>
                            </div>
                          </Link>
                        ))}
                      </div>
                    </motion.div>
                  )}
                </AnimatePresence>
              </div>

              {/* Pricing Link */}
              <Link
                to="/pricing"
                className={cn(
                  'block py-2 px-3 rounded-lg text-base font-medium transition-colors',
                  isActiveLink('/pricing')
                    ? 'text-primary bg-primary/5'
                    : 'text-asgard-600 dark:text-asgard-400 hover:bg-asgard-50 dark:hover:bg-asgard-900'
                )}
              >
                Pricing
              </Link>

              {/* About Link */}
              <Link
                to="/about"
                className={cn(
                  'block py-2 px-3 rounded-lg text-base font-medium transition-colors',
                  isActiveLink('/about')
                    ? 'text-primary bg-primary/5'
                    : 'text-asgard-600 dark:text-asgard-400 hover:bg-asgard-50 dark:hover:bg-asgard-900'
                )}
              >
                About
              </Link>

              {/* Contact Link */}
              <Link
                to="/contact"
                className={cn(
                  'block py-2 px-3 rounded-lg text-base font-medium transition-colors',
                  isActiveLink('/contact')
                    ? 'text-primary bg-primary/5'
                    : 'text-asgard-600 dark:text-asgard-400 hover:bg-asgard-50 dark:hover:bg-asgard-900'
                )}
              >
                Contact
              </Link>

              {/* Auth Buttons for Mobile */}
              {!isAuthenticated && (
                <div className="pt-4 mt-4 flex flex-col gap-2 border-t border-asgard-100 dark:border-asgard-800">
                  <Link to="/signin">
                    <Button variant="outline" className="w-full">
                      Sign In
                    </Button>
                  </Link>
                  <Link to="/signup">
                    <Button className="w-full">Get Started</Button>
                  </Link>
                </div>
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </header>
  );
}
