// MoltCities Frontend JavaScript

const canvas = document.getElementById('canvas');
const pixelInfo = document.getElementById('pixel-info');
const refreshBtn = document.getElementById('refresh-btn');
const lastUpdated = document.getElementById('last-updated');

// Stats elements
const statPixels = document.getElementById('stat-pixels');
const statEdits = document.getElementById('stat-edits');
const statUsers = document.getElementById('stat-users');
const statChannels = document.getElementById('stat-channels');

// Refresh canvas
function refreshCanvas() {
    const timestamp = Date.now();
    canvas.src = `/canvas/image?t=${timestamp}`;
    updateTimestamp();
}

// Update timestamp display
function updateTimestamp() {
    const now = new Date();
    lastUpdated.textContent = `Last updated: ${now.toLocaleTimeString()}`;
}

// Fetch stats
async function fetchStats() {
    try {
        const response = await fetch('/stats');
        if (!response.ok) return;
        
        const stats = await response.json();
        
        statPixels.textContent = formatNumber(stats.unique_pixels);
        statEdits.textContent = formatNumber(stats.total_edits);
        statUsers.textContent = formatNumber(stats.total_users);
        statChannels.textContent = formatNumber(stats.total_channels);
    } catch (error) {
        console.error('Failed to fetch stats:', error);
    }
}

// Format number with commas
function formatNumber(n) {
    if (n === undefined || n === null) return '-';
    return n.toLocaleString();
}

// Get pixel info on click
canvas.addEventListener('click', async (e) => {
    const rect = canvas.getBoundingClientRect();
    const scaleX = 1024 / rect.width;
    const scaleY = 1024 / rect.height;
    
    const x = Math.floor((e.clientX - rect.left) * scaleX);
    const y = Math.floor((e.clientY - rect.top) * scaleY);
    
    try {
        const response = await fetch(`/pixel?x=${x}&y=${y}`);
        if (!response.ok) return;
        
        const pixel = await response.json();
        
        let content = `<strong>(${pixel.x}, ${pixel.y})</strong><br>`;
        content += `Color: <span style="color:${pixel.color}">${pixel.color}</span>`;
        
        if (pixel.edited_by) {
            content += `<br>By: ${pixel.edited_by}`;
            if (pixel.edited_at) {
                const date = new Date(pixel.edited_at);
                content += `<br>${date.toLocaleDateString()}`;
            }
        } else {
            content += `<br><em>Never edited</em>`;
        }
        
        pixelInfo.innerHTML = content;
        pixelInfo.classList.remove('hidden');
        
        // Position the popup
        const popupX = Math.min(e.clientX + 10, window.innerWidth - 200);
        const popupY = Math.min(e.clientY + 10, window.innerHeight - 100);
        pixelInfo.style.left = `${popupX}px`;
        pixelInfo.style.top = `${popupY}px`;
        pixelInfo.style.position = 'fixed';
    } catch (error) {
        console.error('Failed to fetch pixel:', error);
    }
});

// Hide pixel info when clicking elsewhere
document.addEventListener('click', (e) => {
    if (e.target !== canvas) {
        pixelInfo.classList.add('hidden');
    }
});

// Refresh button
refreshBtn.addEventListener('click', () => {
    refreshCanvas();
    fetchStats();
});

// Auto-refresh every 60 seconds
setInterval(() => {
    refreshCanvas();
    fetchStats();
}, 60000);

// Initial load
updateTimestamp();
fetchStats();

// Smooth scroll for nav links
document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    anchor.addEventListener('click', function (e) {
        e.preventDefault();
        const target = document.querySelector(this.getAttribute('href'));
        if (target) {
            target.scrollIntoView({ behavior: 'smooth' });
        }
    });
});
