// Smooth scrolling for navigation links
document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    anchor.addEventListener('click', function (e) {
        e.preventDefault();
        const target = document.querySelector(this.getAttribute('href'));
        if (target) {
            target.scrollIntoView({
                behavior: 'smooth',
                block: 'start'
            });
        }
    });
});

// Intersection Observer for fade-in animations
const observerOptions = {
    threshold: 0.1,
    rootMargin: '0px 0px -50px 0px'
};

const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
        if (entry.isIntersecting) {
            entry.target.classList.add('animate-in');
        }
    });
}, observerOptions);

// Observe elements for animation
document.addEventListener('DOMContentLoaded', () => {
    const animateElements = document.querySelectorAll('.feature-card, .eco-card, .arch-layer');
    animateElements.forEach(el => observer.observe(el));
});

// Add CSS for animations
const style = document.createElement('style');
style.textContent = `
    .feature-card, .eco-card, .arch-layer {
        opacity: 0;
        transform: translateY(30px);
        transition: opacity 0.6s ease, transform 0.6s ease;
    }

    .animate-in {
        opacity: 1;
        transform: translateY(0);
    }
`;
document.head.appendChild(style);

// Header scroll effect
window.addEventListener('scroll', () => {
    const header = document.querySelector('header');
    if (window.scrollY > 100) {
        header.style.background = 'rgba(255, 255, 255, 0.98)';
        header.style.boxShadow = '0 2px 20px rgba(0, 0, 0, 0.1)';
    } else {
        header.style.background = 'rgba(255, 255, 255, 0.95)';
        header.style.boxShadow = 'none';
    }
});

// Typing effect for hero text (optional)
function typeWriter(element, text, speed = 50) {
    let i = 0;
    element.innerHTML = '';
    function type() {
        if (i < text.length) {
            element.innerHTML += text.charAt(i);
            i++;
            setTimeout(type, speed);
        }
    }
    type();
}

// Uncomment to add typing effect to hero title
// document.addEventListener('DOMContentLoaded', () => {
//     const heroTitle = document.querySelector('.hero h1');
//     const originalText = heroTitle.textContent;
//     typeWriter(heroTitle, originalText, 100);
// });

// Download functionality
document.addEventListener('DOMContentLoaded', () => {
    // Checksum data - In production, these would be fetched from server
    const checksums = {
        windows: 'a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890',
        macos: 'b2c3d4e5f678901234567890123456789012345678901234567890123456789012',
        linux: 'c3d4e5f6789012345678901234567890123456789012345678901234567890123'
    };

    // Populate checksums
    Object.keys(checksums).forEach(platform => {
        const checksumEl = document.getElementById(`${platform}-checksum`);
        if (checksumEl) {
            checksumEl.textContent = checksums[platform];
        }
    });

    // Copy checksum functionality
    document.querySelectorAll('.copy-checksum').forEach(button => {
        button.addEventListener('click', (e) => {
            const platform = e.target.dataset.checksum;
            const checksum = checksums[platform];
            navigator.clipboard.writeText(checksum).then(() => {
                const originalText = e.target.textContent;
                e.target.textContent = 'Copied!';
                e.target.style.background = '#28a745';
                e.target.style.color = 'white';
                setTimeout(() => {
                    e.target.textContent = originalText;
                    e.target.style.background = '';
                    e.target.style.color = '';
                }, 2000);
            });
        });
    });

    // Download tracking (basic analytics)
    document.querySelectorAll('.download-btn').forEach(button => {
        button.addEventListener('click', (e) => {
            const platform = e.target.dataset.platform || e.target.closest('a').dataset.platform;
            console.log(`Download started for ${platform} platform`);

            // In production, send analytics event
            // gtag('event', 'download', {
            //     'event_category': 'engagement',
            //     'event_label': platform
            // });
        });
    });

    // Release toggle (for development - remove in production)
    const releaseToggle = document.createElement('div');
    releaseToggle.className = 'release-toggle';
    releaseToggle.innerHTML = `
        <label for="release-toggle">Show Download Section (Dev Mode):</label>
        <input type="checkbox" id="release-toggle">
    `;

    const installationCenter = document.querySelector('.installation-center .container');
    if (installationCenter) {
        installationCenter.insertBefore(releaseToggle, installationCenter.firstChild.nextSibling);

        const toggle = document.getElementById('release-toggle');
        const comingSoonSection = document.getElementById('coming-soon-section');
        const downloadsSection = document.getElementById('downloads-section');

        toggle.addEventListener('change', (e) => {
            if (e.target.checked) {
                comingSoonSection.style.display = 'none';
                downloadsSection.style.display = 'block';
            } else {
                comingSoonSection.style.display = 'block';
                downloadsSection.style.display = 'none';
            }
        });
    }
});

// File validation helper
function validateFile(file, expectedChecksum) {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = async (e) => {
            const buffer = e.target.result;
            const hashBuffer = await crypto.subtle.digest('SHA-256', buffer);
            const hashArray = Array.from(new Uint8Array(hashBuffer));
            const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');

            if (hashHex === expectedChecksum) {
                resolve(true);
            } else {
                resolve(false);
            }
        };
        reader.onerror = reject;
        reader.readAsArrayBuffer(file);
    });
}
