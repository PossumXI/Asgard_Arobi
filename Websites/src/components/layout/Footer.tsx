import { Link } from 'react-router-dom';
import { Github, Twitter, Linkedin, Mail, Globe } from 'lucide-react';

const footerLinks = {
  asgard: [
    { label: 'Valkyrie', href: '/valkyrie' },
    { label: 'Pricilla', href: '/pricilla' },
    { label: 'Giru', href: '/giru' },
  ],
  auraGenesis: [
    { label: 'APEX-OS-LQ', href: '/apex-os' },
    { label: 'Foundation', href: '/foundation' },
    { label: 'ICF Program', href: '/icf' },
  ],
  company: [
    { label: 'About', href: '/about' },
    { label: 'Contact', href: '/contact' },
    { label: 'Pricing', href: '/pricing' },
  ],
  legal: [
    { label: 'Privacy Policy', href: '/privacy' },
    { label: 'Terms of Service', href: '/terms' },
    { label: 'Security', href: '/security' },
  ],
};

const socialLinks = [
  { icon: Twitter, href: 'https://twitter.com/auragenesis', label: 'Twitter' },
  { icon: Github, href: 'https://github.com/aura-genesis', label: 'GitHub' },
  { icon: Linkedin, href: 'https://www.linkedin.com/company/apisnet', label: 'LinkedIn' },
  { icon: Mail, href: 'mailto:Gaetano@aura-genesis.org', label: 'Email' },
];

export default function Footer() {
  return (
    <footer className="bg-asgard-50 dark:bg-asgard-900/50 border-t border-asgard-100 dark:border-asgard-800">
      <div className="container-wide py-16">
        <div className="grid grid-cols-2 md:grid-cols-5 gap-8">
          {/* Company Info */}
          <div className="col-span-2">
            <Link to="/" className="flex items-center gap-2 mb-4">
              <svg
                viewBox="0 0 36 36"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
                className="w-8 h-8 text-primary"
              >
                <circle cx="18" cy="18" r="16" stroke="currentColor" strokeWidth="2" />
                <path d="M18 6L26 22H10L18 6Z" fill="currentColor" />
                <circle cx="18" cy="28" r="3" fill="currentColor" />
              </svg>
              <span className="font-semibold text-lg text-asgard-900 dark:text-white">
                ASGARD by Arobi
              </span>
            </Link>
            <p className="text-sm text-asgard-500 dark:text-asgard-400 mb-4 max-w-xs">
              Planetary-scale autonomous defense and humanitarian aid system. 
              Protecting humanity through intelligent technology.
            </p>
            <p className="text-sm text-asgard-500 dark:text-asgard-400 mb-2">
              © 2026 Arobi. All Rights Reserved.
            </p>
            <a 
              href="https://aura-genesis.org" 
              target="_blank" 
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1.5 text-sm text-primary hover:text-primary/80 transition-colors"
            >
              <Globe className="w-4 h-4" />
              aura-genesis.org
            </a>
          </div>

          {/* ASGARD Products */}
          <div>
            <h4 className="font-semibold text-sm text-asgard-900 dark:text-white mb-4">
              ASGARD Defense
            </h4>
            <ul className="space-y-3">
              {footerLinks.asgard.map((link) => (
                <li key={link.href}>
                  <Link
                    to={link.href}
                    className="text-sm text-asgard-500 hover:text-asgard-900 dark:text-asgard-400 dark:hover:text-white transition-colors"
                  >
                    {link.label}
                  </Link>
                </li>
              ))}
            </ul>
            <h4 className="font-semibold text-sm text-asgard-900 dark:text-white mb-4 mt-6">
              Aura Genesis
            </h4>
            <ul className="space-y-3">
              {footerLinks.auraGenesis.map((link) => (
                <li key={link.href}>
                  <Link
                    to={link.href}
                    className="text-sm text-asgard-500 hover:text-asgard-900 dark:text-asgard-400 dark:hover:text-white transition-colors"
                  >
                    {link.label}
                  </Link>
                </li>
              ))}
            </ul>
          </div>

          {/* Company */}
          <div>
            <h4 className="font-semibold text-sm text-asgard-900 dark:text-white mb-4">
              Company
            </h4>
            <ul className="space-y-3">
              {footerLinks.company.map((link) => (
                <li key={link.href}>
                  <Link
                    to={link.href}
                    className="text-sm text-asgard-500 hover:text-asgard-900 dark:text-asgard-400 dark:hover:text-white transition-colors"
                  >
                    {link.label}
                  </Link>
                </li>
              ))}
            </ul>
          </div>

          {/* Legal */}
          <div>
            <h4 className="font-semibold text-sm text-asgard-900 dark:text-white mb-4">
              Legal
            </h4>
            <ul className="space-y-3">
              {footerLinks.legal.map((link) => (
                <li key={link.href}>
                  <Link
                    to={link.href}
                    className="text-sm text-asgard-500 hover:text-asgard-900 dark:text-asgard-400 dark:hover:text-white transition-colors"
                  >
                    {link.label}
                  </Link>
                </li>
              ))}
            </ul>
          </div>
        </div>

        {/* Social Links */}
        <div className="mt-10 pt-8 border-t border-asgard-200 dark:border-asgard-800">
          <div className="flex flex-col sm:flex-row items-center justify-between gap-6">
            <div className="flex items-center gap-3">
              {socialLinks.map((social) => (
                <a
                  key={social.label}
                  href={social.href}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="p-2 rounded-lg text-asgard-400 hover:text-asgard-900 hover:bg-asgard-100 dark:hover:text-white dark:hover:bg-asgard-800 transition-colors"
                  aria-label={social.label}
                >
                  <social.icon className="w-5 h-5" />
                </a>
              ))}
            </div>
            <span className="flex items-center gap-1.5 text-xs text-asgard-400">
              <span className="w-2 h-2 rounded-full bg-success animate-pulse" />
              All systems operational
            </span>
          </div>
        </div>

        {/* Bottom Bar */}
        <div className="mt-8 pt-8 border-t border-asgard-200 dark:border-asgard-800 flex flex-col sm:flex-row items-center justify-between gap-4">
          <p className="text-sm text-asgard-500 dark:text-asgard-400">
            All Rights Reserved By. <span className="font-medium text-asgard-700 dark:text-asgard-300">Arobi ©2025-2026</span>
          </p>
          <a 
            href="mailto:Gaetano@aura-genesis.org"
            className="text-sm text-asgard-500 hover:text-primary dark:text-asgard-400 dark:hover:text-primary transition-colors"
          >
            Contact: Gaetano@aura-genesis.org
          </a>
        </div>
      </div>
    </footer>
  );
}
