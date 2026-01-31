# Social Media Images Troubleshooting Guide

## Issue: Images Not Loading When Sharing Links

If your social media images aren't showing up when you share links, follow this troubleshooting guide.

## Step 1: Verify Images Exist and Are Accessible

1. **Open the test page**: Visit `https://aura-genesis.org/images/test.html`
2. **Check if images load**: All 4 social images should display on this page
3. **Check browser console**: Open Developer Tools (F12) and look for any errors

## Step 2: Verify File Upload

Ensure your images are uploaded to the correct location on your web server:

```
aura-genesis.org/
├── images/
│   ├── aura-genesis-social.jpg
│   ├── apex-os-lq-social.jpg
│   ├── foundation-social.jpg
│   └── icf-social.jpg
```

## Step 3: Check Image Properties

Your images must meet these specifications:

- **Dimensions**: Exactly 1200x630 pixels (1.91:1 aspect ratio)
- **Format**: JPG (not PNG, GIF, or other formats)
- **File Size**: Under 5MB (preferably under 1MB for faster loading)
- **File Names**: Exact match (case-sensitive)

### How to Check Image Dimensions

**Windows**: Right-click image → Properties → Details tab → Dimensions
**Mac**: Right-click image → Get Info → More Info → Dimensions
**Online**: Use tools like [Image Size Analyzer](https://imagesize.org/)

## Step 4: Test Direct URLs

Try accessing these URLs directly in your browser:

- `https://aura-genesis.org/images/aura-genesis-social.jpg`
- `https://aura-genesis.org/images/apex-os-lq-social.jpg`
- `https://aura-genesis.org/images/foundation-social.jpg`
- `https://aura-genesis.org/images/icf-social.jpg`

If these don't load, the issue is with file upload or server configuration.

## Step 5: Clear Social Media Cache

Social media platforms cache images. After uploading new images:

### Facebook
1. Go to [Facebook Sharing Debugger](https://developers.facebook.com/tools/debug/)
2. Enter your URL (e.g., `https://aura-genesis.org/`)
3. Click "Debug"
4. Click "Scrape Again" if needed

### Twitter/X
1. Go to [Twitter Card Validator](https://cards-dev.twitter.com/validator)
2. Enter your URL
3. Click "Preview Card"

### LinkedIn
1. Go to [LinkedIn Post Inspector](https://www.linkedin.com/post-inspector/)
2. Enter your URL
3. Click "Inspect"

## Step 6: Check Server Configuration

If images still don't load, check with your hosting provider:

### Required Server Features
- **Apache/Nginx**: Modern web server
- **PHP**: Version 7.0+ (if using dynamic features)
- **Image MIME Types**: JPG files served as `image/jpeg`
- **Directory Permissions**: `images/` directory must be readable (755)
- **File Permissions**: Image files must be readable (644)

### Common Server Issues
- **Hotlink Protection**: May block social media crawlers
- **Mod_Security**: May flag image requests as suspicious
- **CDN Configuration**: May not be configured for your domain

## Step 7: Regenerate Images

If all else fails, recreate the images:

1. **Use the templates**: Open `apex-os-lq-social.html` and `aura-genesis-social.html` in your browser
2. **Recreate in design tool**: Use Canva, Figma, or Photoshop to recreate the designs
3. **Save with correct specifications**:
   - 1200x630px
   - JPG format
   - High quality (80-90%)
   - RGB color mode
4. **Upload again** and test

## Step 8: Alternative Solutions

### Option 1: Use External Image Hosting
If your server won't serve images properly:
- Upload to Imgur, Cloudinary, or similar service
- Update meta tags to use the new URLs
- Ensure the service allows hotlinking

### Option 2: Use Data URIs (Not Recommended)
- Convert images to base64 data URIs
- Embed directly in meta tags
- Increases page size significantly

### Option 3: Use SVG Social Images
- Create vector versions of your social images
- Serve as SVG (supported by most platforms)
- Smaller file size and scalable

## Quick Diagnostic Checklist

- [ ] Images uploaded to correct directory
- [ ] File names exactly match meta tags
- [ ] Images are 1200x630px
- [ ] Images are JPG format
- [ ] Direct URLs work in browser
- [ ] Test page shows images
- [ ] Social media caches cleared
- [ ] Server allows image hotlinking
- [ ] No server-side blocking rules

## Need Help?

If you continue having issues:

1. **Share the test page results**: What do you see at `https://aura-genesis.org/images/test.html`?
2. **Check browser network tab**: Look for failed image requests
3. **Contact hosting support**: Ask about image serving and hotlinking policies
4. **Try different image formats**: Some servers have issues with certain image types

## Prevention Tips

- **Test before major sharing**: Always verify images work before important social media campaigns
- **Keep backups**: Store original design files for easy recreation
- **Monitor regularly**: Check that images still load periodically
- **Use consistent naming**: Stick to the naming convention established
