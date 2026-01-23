import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { Home, ArrowLeft, Search } from 'lucide-react';
import { Button } from '@/components/ui/Button';

export default function NotFound() {
  return (
    <div className="min-h-screen flex items-center justify-center pt-16 pb-12 px-4">
      <div className="absolute inset-0 -z-10">
        <div className="absolute inset-0 bg-gradient-to-b from-asgard-50 via-white to-white dark:from-asgard-950 dark:via-asgard-950 dark:to-asgard-950" />
        <div className="absolute top-1/3 left-1/2 -translate-x-1/2 w-[600px] h-[400px] bg-primary/5 rounded-full blur-3xl" />
      </div>

      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="text-center max-w-md"
      >
        {/* 404 Illustration */}
        <div className="mb-8">
          <svg
            viewBox="0 0 200 120"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            className="w-48 h-28 mx-auto text-primary"
          >
            <motion.circle
              cx="100"
              cy="60"
              r="45"
              stroke="currentColor"
              strokeWidth="2"
              fill="none"
              strokeDasharray="4 4"
              initial={{ rotate: 0 }}
              animate={{ rotate: 360 }}
              transition={{ duration: 60, repeat: Infinity, ease: 'linear' }}
            />
            <motion.circle
              cx="100"
              cy="60"
              r="30"
              stroke="currentColor"
              strokeWidth="2"
              fill="none"
              initial={{ scale: 0.9, opacity: 0.5 }}
              animate={{ scale: 1.1, opacity: 1 }}
              transition={{ duration: 2, repeat: Infinity, repeatType: 'reverse' }}
            />
            <text
              x="100"
              y="68"
              textAnchor="middle"
              className="fill-current font-bold text-2xl"
            >
              404
            </text>
            {/* Satellite orbiting */}
            <motion.g
              initial={{ rotate: 0 }}
              animate={{ rotate: 360 }}
              transition={{ duration: 10, repeat: Infinity, ease: 'linear' }}
              style={{ transformOrigin: '100px 60px' }}
            >
              <rect
                x="140"
                y="55"
                width="10"
                height="10"
                fill="currentColor"
                opacity="0.6"
              />
            </motion.g>
          </svg>
        </div>

        <h1 className="text-display text-asgard-900 dark:text-white mb-4">
          Signal Lost
        </h1>
        
        <p className="text-body-lg text-asgard-500 dark:text-asgard-400 mb-8">
          The page you're looking for seems to have drifted out of orbit. 
          Let's get you back on course.
        </p>

        <div className="flex flex-col sm:flex-row items-center justify-center gap-3">
          <Link to="/">
            <Button size="lg" className="group">
              <Home className="w-4 h-4 mr-2" />
              Back to Home
            </Button>
          </Link>
          <Button 
            variant="outline" 
            size="lg"
            onClick={() => window.history.back()}
          >
            <ArrowLeft className="w-4 h-4 mr-2" />
            Go Back
          </Button>
        </div>

        <div className="mt-12 pt-8 border-t border-asgard-200 dark:border-asgard-800">
          <p className="text-sm text-asgard-400 mb-4">
            Looking for something specific?
          </p>
          <div className="flex gap-4 justify-center text-sm">
            <Link 
              to="/features" 
              className="text-primary hover:text-primary-600 transition-colors"
            >
              Features
            </Link>
            <Link 
              to="/pricing" 
              className="text-primary hover:text-primary-600 transition-colors"
            >
              Pricing
            </Link>
            <Link 
              to="/about" 
              className="text-primary hover:text-primary-600 transition-colors"
            >
              About
            </Link>
            <Link 
              to="/gov" 
              className="text-primary hover:text-primary-600 transition-colors"
            >
              Government
            </Link>
          </div>
        </div>
      </motion.div>
    </div>
  );
}
