# Aura Genesis Website

A professional, sleek website for Aura Genesis - Arobi Technology Alliance Inc., showcasing APEX-OS-LQ, the revolutionary autonomous computer operating system.

## Features

- **Modern Design**: Glassmorphic effects, white and black color scheme with autonomous flow
- **Responsive**: Optimized for desktop, tablet, and mobile devices
- **Macintosh-inspired**: Subtle gradients, rounded corners, and professional aesthetics
- **Multi-page**: Home, APEX-OS-LQ download page, Foundation, and ICF program pages

## Pages

- `index.html` - Landing page with hero section, features, and ecosystem overview
- `apex-os-lq.html` - Coming soon installation center with pricing and platform support
- `foundation.html` - Aura Genesis Foundation page with governance and developer programs
- `icf.html` - Intelligence Consule Federation page with PoI consensus details

## Technologies

- HTML5
- CSS3 (with glassmorphic effects and animations)
- JavaScript (smooth scrolling, fade-in animations, download functionality)
- Font Awesome icons
- PHP API for download management

## Download Infrastructure

When APEX-OS-LQ is ready for release:

### Directory Structure
```
downloads/
├── manifest.json          # Package metadata and checksums
├── README.md             # Download instructions
├── packages/             # Actual installation files
│   ├── apex-os-lq-windows-1.0.0.exe
│   ├── apex-os-lq-macos-1.0.0.dmg
│   ├── apex-os-lq-linux-1.0.0.AppImage
│   └── apex-os-lq-usb-creator-1.0.0.exe
api/
└── downloads.php         # API endpoint for download info
```

### Deployment Steps

1. **Prepare Packages**: Build installation packages for each platform
2. **Generate Checksums**: Run `./deploy-downloads.sh update-checksums`
3. **Verify Integrity**: Run `./deploy-downloads.sh verify`
4. **Deploy**: Run `./deploy-downloads.sh deploy`
5. **Enable Downloads**: Remove the dev toggle in `apex-os-lq.html`

### API Endpoints

- `GET /api/downloads` - Full manifest data
- `GET /api/downloads/platforms` - Available platforms
- `GET /api/downloads/latest` - Latest version info
- `GET /api/downloads/{platform}` - Platform-specific info

## Social Media Integration

All pages include comprehensive social sharing meta tags:

### Open Graph (Facebook/LinkedIn)
- Custom titles and descriptions for each page
- Dedicated social media images (1200x630px)
- Proper image dimensions and alt text
- Site branding and company information

### Twitter Cards
- Large image card format for maximum visibility
- Optimized titles and descriptions
- Custom Twitter images with proper dimensions

### SEO Optimization
- Descriptive page titles and meta descriptions
- Relevant keywords for search engines
- Author and theme color information

### Social Images Required
- `images/aura-genesis-social.jpg` - Main homepage
- `images/apex-os-lq-social.jpg` - APEX-OS-LQ page
- `images/foundation-social.jpg` - Foundation page
- `images/icf-social.jpg` - ICF page

View the HTML templates in the `images/` directory for design specifications.

## Design Philosophy

The website embodies a "new/old Macintosh window" feel with:
- Clean, professional typography
- Subtle gradients and shadows
- Rounded corners and smooth transitions
- Glassmorphic cards and elements
- High contrast white and black palette
- Security-focused download verification

## Security Features

- SHA256 checksum verification for all downloads
- Package integrity validation
- Secure headers in .htaccess
- API rate limiting (to be implemented)
- Download analytics tracking

## Contact

- Email: contact@aura-genesis.org
- Discord: https://discord.gg/h64Cg8c6
- Company: Arobi Technology Alliance Inc.

## Copyright

© 2025 Arobi Technology Alliance Inc. All rights reserved.
