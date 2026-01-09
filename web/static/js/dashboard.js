// Dashboard JavaScript utilities

/**
 * Format bytes to human readable format
 */
function formatBytes(bytes, decimals = 2) {
  if (bytes === 0) return '0 Bytes';
  if (bytes === 'N/A') return 'N/A';

  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];

  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

/**
 * Format CPU usage
 */
function formatCPU(cpu) {
  if (cpu === 'N/A') return 'N/A';
  return cpu;
}

/**
 * Format timestamp to readable date
 */
function formatTimestamp(timestamp) {
  if (!timestamp) return 'N/A';
  const date = new Date(timestamp);
  return date.toLocaleString();
}

/**
 * Escape HTML to prevent XSS
 */
function escapeHtml(text) {
  if (typeof text !== 'string') return text;
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

/**
 * Debounce function to limit function calls
 */
function debounce(func, wait) {
  let timeout;
  return function executedFunction(...args) {
    const later = () => {
      clearTimeout(timeout);
      func(...args);
    };
    clearTimeout(timeout);
    timeout = setTimeout(later, wait);
  };
}

/**
 * Show notification/toast message
 */
function showNotification(message, type = 'info') {
  // Create notification element
  const notification = document.createElement('div');
  notification.className = `notification notification-${type}`;
  notification.textContent = message;
  notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        padding: 1rem 1.5rem;
        background-color: ${type === 'error' ? '#dc3545' : type === 'success' ? '#28a745' : '#007bff'};
        color: white;
        border-radius: 4px;
        box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        z-index: 1000;
        animation: slideIn 0.3s ease-out;
    `;

  document.body.appendChild(notification);

  // Remove after 3 seconds
  setTimeout(() => {
    notification.style.animation = 'slideOut 0.3s ease-out';
    setTimeout(() => {
      document.body.removeChild(notification);
    }, 300);
  }, 3000);
}

/**
 * Copy text to clipboard
 */
async function copyToClipboard(text) {
  try {
    await navigator.clipboard.writeText(text);
    showNotification('Copied to clipboard', 'success');
  } catch (err) {
    console.error('Failed to copy:', err);
    showNotification('Failed to copy to clipboard', 'error');
  }
}

/**
 * Filter logs by search term
 */
function filterLogs(logs, searchTerm) {
  if (!searchTerm) return logs;
  const term = searchTerm.toLowerCase();
  return logs.filter(log => log.toLowerCase().includes(term));
}

/**
 * Filter logs by log level
 */
function filterLogsByLevel(logs, level) {
  if (!level || level === 'all') return logs;
  const levelUpper = level.toUpperCase();
  return logs.filter(log => {
    const logUpper = log.toUpperCase();
    return logUpper.includes(`[${levelUpper}]`) ||
      logUpper.startsWith(levelUpper) ||
      logUpper.includes(` ${levelUpper} `);
  });
}

/**
 * Format log line with syntax highlighting
 */
function formatLogLine(line) {
  // Add basic syntax highlighting for common log patterns
  let formatted = escapeHtml(line);

  // Highlight ERROR, WARN, INFO, DEBUG
  formatted = formatted.replace(/\b(ERROR|FATAL)\b/gi, '<span style="color: #dc3545; font-weight: bold;">$1</span>');
  formatted = formatted.replace(/\b(WARN|WARNING)\b/gi, '<span style="color: #ffc107; font-weight: bold;">$1</span>');
  formatted = formatted.replace(/\b(INFO)\b/gi, '<span style="color: #17a2b8; font-weight: bold;">$1</span>');
  formatted = formatted.replace(/\b(DEBUG|TRACE)\b/gi, '<span style="color: #6c757d; font-weight: bold;">$1</span>');

  // Highlight timestamps
  formatted = formatted.replace(/(\d{4}-\d{2}-\d{2}[\sT]\d{2}:\d{2}:\d{2})/g, '<span style="color: #6c757d;">$1</span>');

  return formatted;
}

/**
 * Auto-scroll log viewer to bottom
 */
function autoScrollLogViewer(element) {
  if (element) {
    element.scrollTop = element.scrollHeight;
  }
}

/**
 * Initialize tooltips or other UI enhancements
 */
function initializeUI() {
  // Add click handlers for copy buttons if they exist
  document.querySelectorAll('.copy-btn').forEach(btn => {
    btn.addEventListener('click', function () {
      const text = this.getAttribute('data-copy');
      if (text) {
        copyToClipboard(text);
      }
    });
  });

  // Add search functionality for tables
  const searchInputs = document.querySelectorAll('.table-search');
  searchInputs.forEach(input => {
    input.addEventListener('input', debounce(function () {
      const searchTerm = this.value.toLowerCase();
      const table = this.closest('.card').querySelector('table');
      if (table) {
        const rows = table.querySelectorAll('tbody tr');
        rows.forEach(row => {
          const text = row.textContent.toLowerCase();
          row.style.display = text.includes(searchTerm) ? '' : 'none';
        });
      }
    }, 300));
  });
}

// Initialize UI when DOM is ready
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initializeUI);
} else {
  initializeUI();
}

// Export functions for use in other scripts
window.dashboardUtils = {
  formatBytes,
  formatCPU,
  formatTimestamp,
  escapeHtml,
  debounce,
  showNotification,
  copyToClipboard,
  filterLogs,
  filterLogsByLevel,
  formatLogLine,
  autoScrollLogViewer
};

