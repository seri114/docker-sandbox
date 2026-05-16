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
    runButton.disabled = running;
    const btnText = runButton.querySelector('.btn-text');
    const btnSpinner = runButton.querySelector('.btn-spinner');

    if (running) {
        btnText.textContent = 'Running...';
        btnSpinner.hidden = false;
        outputElement.classList.remove('output-empty');
    } else {
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

// Stop SSE connection
function stopExecution() {
    if (eventSource) {
        eventSource.close();
        eventSource = null;
    }
    setRunningState(false);
}

// Execute code
async function executeCode() {
    if (isRunning) {
        stopExecution();
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

        // Start SSE stream
        eventSource = new EventSource(`${API_BASE}/api/execute/stream?execution_id=${executionId}`);

        eventSource.onmessage = (event) => {
            const data = event.data;
            if (data === '[DONE]') {
                // Execution completed
                stopExecution();
                clearButton.hidden = false;
                return;
            }
            appendOutput(data);
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
    stopExecution();
});
