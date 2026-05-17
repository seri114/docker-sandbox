// Python Sandbox Web UI

const API_BASE = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
    ? 'http://localhost:8000'
    : '';

// DOM Elements
const codeTextarea = document.getElementById('code');
const timeoutInput = document.getElementById('timeout');
const runButton = document.getElementById('run');
const outputElement = document.getElementById('output');
const clearButton = document.getElementById('clear');

// State
let eventSource = null;
let isRunning = false;
let currentExecutionId = null;

// Validate timeout input (1-60 seconds)
function validateTimeout() {
    let value = parseInt(timeoutInput.value, 10);
    if (isNaN(value) || value < 1) value = 1;
    if (value > 60) value = 60;
    timeoutInput.value = value;
    return value;
}

timeoutInput.addEventListener('change', validateTimeout);
timeoutInput.addEventListener('blur', validateTimeout);

// Update UI state during execution
function setRunningState(running) {
    isRunning = running;
    const btnText = runButton.querySelector('.btn-text');
    const btnSpinner = runButton.querySelector('.btn-spinner');

    if (running) {
        runButton.classList.add('stop-btn');
        btnText.textContent = 'Stop';
        btnSpinner.hidden = true;
        outputElement.classList.remove('output-empty');
    } else {
        runButton.classList.remove('stop-btn');
        btnText.textContent = 'Run Code';
        btnSpinner.hidden = true;
    }
}

// Append output to the result area
function appendOutput(text) {
    outputElement.textContent += text;
    // Auto-scroll to bottom
    outputElement.scrollTop = outputElement.scrollHeight;
}

// Clear output
function clearOutput() {
    outputElement.textContent = '';
    outputElement.classList.add('output-empty');
    clearButton.hidden = true;
}

clearButton.addEventListener('click', clearOutput);

// Stop SSE connection and cancel execution
async function stopExecution(cancelled = false) {
    if (eventSource) {
        eventSource.close();
        eventSource = null;
    }

    // Cancel the execution on the server (only if user cancelled)
    if (cancelled && currentExecutionId) {
        try {
            await fetch(`${API_BASE}/api/execute/cancel`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ execution_id: currentExecutionId })
            });
            appendOutput('\n--- Execution cancelled ---');
        } catch (error) {
            console.error('Cancel error:', error);
        }
        currentExecutionId = null;
    } else {
        // Normal completion, just clear the execution ID
        currentExecutionId = null;
    }

    setRunningState(false);
}

// Execute code
async function executeCode() {
    if (isRunning) {
        stopExecution(true);
        return;
    }

    const code = codeTextarea.value.trim();
    if (!code) {
        alert('Please enter some Python code to execute.');
        return;
    }

    const timeout = validateTimeout();

    // Clear previous output
    clearOutput();
    outputElement.textContent = '';
    outputElement.classList.remove('output-empty');

    setRunningState(true);

    try {
        // Create execution
        const response = await fetch(`${API_BASE}/api/execute`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                code: code,
                timeout: timeout
            })
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();
        const executionId = data.execution_id;
        currentExecutionId = executionId;

        // Start SSE stream
        eventSource = new EventSource(`${API_BASE}/api/execute/stream?execution_id=${executionId}`);

        eventSource.onmessage = (event) => {
            const data = event.data;
            // Parse JSON to extract the actual output
            try {
                const msg = JSON.parse(data);
                if (msg.data) {
                    appendOutput(msg.data);
                }
            } catch {
                // Not JSON, append as-is (handles [DONE])
                if (data === '[DONE]') {
                    stopExecution();
                    clearButton.hidden = false;
                } else {
                    appendOutput(data);
                }
            }
        };

        eventSource.onerror = (error) => {
            console.error('SSE error:', error);
            appendOutput('\n--- Connection error ---');
            stopExecution();
            clearButton.hidden = false;
        };

        eventSource.onopen = () => {
            // Connection established
            console.log('SSE connection opened');
        };

    } catch (error) {
        console.error('Execution error:', error);
        appendOutput(`Error: ${error.message}`);
        stopExecution();
        clearButton.hidden = false;
    }
}

// Event listeners
runButton.addEventListener('click', executeCode);

// Keyboard shortcut: Ctrl+Enter / Cmd+Enter to run
codeTextarea.addEventListener('keydown', (event) => {
    if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
        event.preventDefault();
        executeCode();
    }
});

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
    stopExecution(true);
});
