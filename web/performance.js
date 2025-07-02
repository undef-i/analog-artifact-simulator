let worker = null;
let wasmReady = false;
let currentImageData = null;
let testResults = {
    components: {},
    scaling: {},
    memory: {}
};

function initWorker() {
    worker = new Worker('worker.js');

    worker.onmessage = function (event) {
        const data = event.data;
        if (data.type === 'wasmLoaded') {
            wasmReady = true;
            updateStatus('WebAssembly module loaded - Ready for bottleneck analysis!');
        } else if (data.type === 'result') {
            // Handle test result
        } else if (data.type === 'error') {
            updateStatus('Error: ' + data.message);
        }
    };

    worker.onerror = function (error) {
        updateStatus('Web Worker encountered an error');
    };
}

function updateStatus(message) {
    const statusDiv = document.getElementById('wasmStatus');
    statusDiv.textContent = message;
}

function generateTestImage(width, height) {
    const canvas = document.createElement('canvas');
    canvas.width = width;
    canvas.height = height;
    const ctx = canvas.getContext('2d');

    const imageData = ctx.createImageData(width, height);
    for (let i = 0; i < imageData.data.length; i += 4) {
        const x = (i / 4) % width;
        const y = Math.floor((i / 4) / width);
        const hue = (x / width) * 360;
        const sat = 0.8;
        const val = 0.9;
        const rgb = hsvToRgb(hue, sat, val);

        imageData.data[i] = rgb.r;
        imageData.data[i + 1] = rgb.g;
        imageData.data[i + 2] = rgb.b;
        imageData.data[i + 3] = 255;
    }

    ctx.putImageData(imageData, 0, 0);
    currentImageData = canvas.toDataURL();

    const testImage = document.getElementById('testImage');
    testImage.src = currentImageData;
    testImage.style.display = 'block';

    return currentImageData;
}

function hsvToRgb(h, s, v) {
    const c = v * s;
    const x = c * (1 - Math.abs((h / 60) % 2 - 1));
    const m = v - c;
    let r, g, b;

    if (h >= 0 && h < 60) {
        r = c; g = x; b = 0;
    } else if (h >= 60 && h < 120) {
        r = x; g = c; b = 0;
    } else if (h >= 120 && h < 180) {
        r = 0; g = c; b = x;
    } else if (h >= 180 && h < 240) {
        r = 0; g = x; b = c;
    } else if (h >= 240 && h < 300) {
        r = x; g = 0; b = c;
    } else {
        r = c; g = 0; b = x;
    }

    return {
        r: Math.round((r + m) * 255),
        g: Math.round((g + m) * 255),
        b: Math.round((b + m) * 255)
    };
}

function scrollToBottom(elementId) {
    const element = document.getElementById(elementId);
    if (element) {
        element.scrollTop = element.scrollHeight;
    }
}

function appendResult(elementId, text) {
    const element = document.getElementById(elementId);
    if (element) {
        element.textContent += text;
        scrollToBottom(elementId);
    }
}

async function runComponentTest(component) {
    if (!wasmReady || !currentImageData) {
        alert('Please load WASM module and select test image first');
        return;
    }

    const button = document.getElementById(`btn${component.charAt(0).toUpperCase() + component.slice(1)}`);


    if (!button) {
        alert('Button not found for component: ' + component);
        return;
    }

    button.disabled = true;

    try {


        const iterations = 20;
        const times = [];
        const startTime = performance.now();

        appendResult('componentResults', `\n=== TESTING ${component.toUpperCase()} COMPONENT ===\n`);
        appendResult('componentResults', `Starting ${iterations} iterations...\n`);

        for (let i = 0; i < iterations; i++) {
            const iterationStart = performance.now();

            await new Promise((resolve, reject) => {
                const requestId = Date.now() + Math.random();
                const timeout = setTimeout(() => {
                    worker.removeEventListener('message', handler);
                    reject(new Error('Request timeout'));
                }, 10000);

                const handler = (event) => {
                    if (event.data.requestId === requestId) {
                        clearTimeout(timeout);
                        worker.removeEventListener('message', handler);
                        times.push(performance.now() - iterationStart);
                        resolve();
                    }
                };

                worker.addEventListener('message', handler);
                worker.postMessage({
                    type: 'processImage',
                    imageData: currentImageData,
                    component: component,
                    requestId: requestId
                });
            });


        }

        const totalTime = performance.now() - startTime;
        const avgTime = times.reduce((a, b) => a + b, 0) / times.length;
        const minTime = Math.min(...times);
        const maxTime = Math.max(...times);
        const stdDev = Math.sqrt(times.reduce((sq, n) => sq + Math.pow(n - avgTime, 2), 0) / times.length);
        const throughput = 1000 / avgTime;

        testResults.components[component] = {
            avgTime,
            minTime,
            maxTime,
            stdDev,
            throughput,
            iterations
        };

        const bottleneckLevel = getBottleneckLevel(avgTime);
        const efficiency = getEfficiencyRating(throughput);

        const result = `Component: ${component}\n` +
            `Average Time: ${avgTime.toFixed(2)} ms\n` +
            `Min/Max Time: ${minTime.toFixed(2)} / ${maxTime.toFixed(2)} ms\n` +
            `Standard Deviation: ${stdDev.toFixed(2)} ms\n` +
            `Throughput: ${throughput.toFixed(2)} ops/sec\n` +
            `Bottleneck Level: ${bottleneckLevel}\n` +
            `Efficiency Rating: ${efficiency}\n` +
            `Optimization Priority: ${getOptimizationPriority(avgTime, stdDev)}\n\n`;

        appendResult('componentResults', result);

    } catch (error) {
        appendResult('componentResults', `ERROR in ${component} test: ${error.message}\n\n`);
    } finally {
        button.disabled = false;
    }
}

function getBottleneckLevel(avgTime) {
    if (avgTime > 100) return 'CRITICAL - Major bottleneck detected';
    if (avgTime > 50) return 'HIGH - Significant performance impact';
    if (avgTime > 25) return 'MEDIUM - Moderate performance impact';
    if (avgTime > 10) return 'LOW - Minor performance impact';
    return 'MINIMAL - Well optimized';
}

function getEfficiencyRating(throughput) {
    if (throughput > 100) return 'EXCELLENT - Very efficient';
    if (throughput > 50) return 'GOOD - Reasonably efficient';
    if (throughput > 20) return 'FAIR - Could be improved';
    if (throughput > 10) return 'POOR - Needs optimization';
    return 'CRITICAL - Requires immediate attention';
}

function getOptimizationPriority(avgTime, stdDev) {
    const variability = stdDev / avgTime;
    if (avgTime > 50 && variability > 0.3) return 'URGENT - High time + High variability';
    if (avgTime > 50) return 'HIGH - High processing time';
    if (variability > 0.5) return 'MEDIUM - High variability';
    return 'LOW - Stable performance';
}

async function runScalingTest() {
    if (!wasmReady) {
        alert('Please load WASM module first');
        return;
    }

    const button = document.getElementById('btnScaling');


    button.disabled = true;


    const resolutions = [
        { width: 320, height: 240, name: '320x240' },
        { width: 640, height: 480, name: '640x480' },
        { width: 1280, height: 720, name: '1280x720' },
        { width: 1920, height: 1080, name: '1920x1080' }
    ];

    appendResult('scalingResults', '\n=== RESOLUTION SCALING ANALYSIS ===\n');
    appendResult('scalingResults', 'Testing performance across different resolutions...\n\n');

    for (let i = 0; i < resolutions.length; i++) {
        const res = resolutions[i];
        const testImage = generateTestImage(res.width, res.height);
        const iterations = 10;
        const times = [];

        appendResult('scalingResults', `Testing ${res.name}...\n`);

        for (let j = 0; j < iterations; j++) {
            const iterationStart = performance.now();

            await new Promise((resolve, reject) => {
                const requestId = Date.now() + Math.random();
                const timeout = setTimeout(() => {
                    worker.removeEventListener('message', handler);
                    reject(new Error('Request timeout'));
                }, 10000);

                const handler = (event) => {
                    if (event.data.requestId === requestId) {
                        clearTimeout(timeout);
                        worker.removeEventListener('message', handler);
                        times.push(performance.now() - iterationStart);
                        resolve();
                    }
                };

                worker.addEventListener('message', handler);
                worker.postMessage({
                    type: 'processImage',
                    imageData: testImage,
                    requestId: requestId
                });
            });
        }

        const avgTime = times.reduce((a, b) => a + b, 0) / times.length;
        const pixelCount = res.width * res.height;
        const timePerPixel = avgTime / pixelCount * 1000000; // nanoseconds per pixel
        const scalingEfficiency = getScalingEfficiency(res.width * res.height, avgTime);

        testResults.scaling[res.name] = {
            avgTime,
            pixelCount,
            timePerPixel,
            scalingEfficiency
        };

        const result = `Resolution: ${res.name}\n` +
            `Average Time: ${avgTime.toFixed(2)} ms\n` +
            `Pixel Count: ${pixelCount.toLocaleString()}\n` +
            `Time per Pixel: ${timePerPixel.toFixed(2)} ns/pixel\n` +
            `Scaling Efficiency: ${scalingEfficiency}\n\n`;

        appendResult('scalingResults', result);

    }

    analyzeScalingPattern();
    button.disabled = false;

}

function getScalingEfficiency(pixelCount, avgTime) {
    const expectedLinearTime = (pixelCount / 307200) * 16.67; // 640x480 baseline at 60fps
    const efficiency = (expectedLinearTime / avgTime) * 100;

    if (efficiency > 90) return 'EXCELLENT - Linear scaling';
    if (efficiency > 70) return 'GOOD - Near-linear scaling';
    if (efficiency > 50) return 'FAIR - Sublinear scaling';
    if (efficiency > 30) return 'POOR - Poor scaling';
    return 'CRITICAL - Very poor scaling';
}

function analyzeScalingPattern() {
    const scalingData = Object.values(testResults.scaling);
    if (scalingData.length < 2) return;

    appendResult('scalingResults', '=== SCALING PATTERN ANALYSIS ===\n');

    const timePerPixelValues = scalingData.map(d => d.timePerPixel);
    const avgTimePerPixel = timePerPixelValues.reduce((a, b) => a + b, 0) / timePerPixelValues.length;
    const variance = timePerPixelValues.reduce((sq, n) => sq + Math.pow(n - avgTimePerPixel, 2), 0) / timePerPixelValues.length;
    const consistency = variance < (avgTimePerPixel * 0.1) ? 'CONSISTENT' : 'INCONSISTENT';

    appendResult('scalingResults', `Average Time per Pixel: ${avgTimePerPixel.toFixed(2)} ns/pixel\n`);
    appendResult('scalingResults', `Scaling Consistency: ${consistency}\n`);

    if (consistency === 'INCONSISTENT') {
        appendResult('scalingResults', 'WARNING: Non-linear scaling detected - possible memory or cache issues\n');
    }

    appendResult('scalingResults', '\n');
}

async function runMemoryEfficiencyTest() {
    if (!wasmReady || !currentImageData) {
        alert('Please load WASM module and select test image first');
        return;
    }

    const button = document.getElementById('btnMemoryEff');


    button.disabled = true;


    appendResult('memoryResults', '\n=== MEMORY EFFICIENCY ANALYSIS ===\n');

    if (!performance.memory) {
        appendResult('memoryResults', 'WARNING: Memory API not available in this browser\n');
        appendResult('memoryResults', 'Memory analysis will be limited\n\n');
    }

    const initialMemory = performance.memory ? performance.memory.usedJSHeapSize : 0;
    let peakMemory = initialMemory;
    const memorySnapshots = [];
    const iterations = 15;

    for (let i = 0; i < iterations; i++) {
        const testImage = generateTestImage(1920, 1080);

        await new Promise((resolve, reject) => {
            const requestId = Date.now() + Math.random();
            const timeout = setTimeout(() => {
                worker.removeEventListener('message', handler);
                reject(new Error('Request timeout'));
            }, 10000);

            const handler = (e) => {
                if (e.data.type === 'result' && e.data.requestId === requestId) {
                    clearTimeout(timeout);
                    worker.removeEventListener('message', handler);
                    resolve();
                }
            };

            worker.addEventListener('message', handler);
            worker.postMessage({
                type: 'processImage',
                imageData: testImage,
                requestId: requestId
            });
        });

        if (performance.memory) {
            const currentMemory = performance.memory.usedJSHeapSize;
            peakMemory = Math.max(peakMemory, currentMemory);
            memorySnapshots.push(currentMemory);
        }


    }

    const finalMemory = performance.memory ? performance.memory.usedJSHeapSize : 0;
    const memoryLeak = finalMemory - initialMemory;
    const avgMemoryUsage = memorySnapshots.reduce((a, b) => a + b, 0) / memorySnapshots.length;
    const memoryEfficiency = calculateMemoryEfficiency(peakMemory - initialMemory);
    const leakSeverity = getLeakSeverity(memoryLeak);

    testResults.memory = {
        initialMemory,
        peakMemory,
        finalMemory,
        memoryLeak,
        avgMemoryUsage,
        memoryEfficiency,
        leakSeverity
    };

    const result = `Initial Memory: ${(initialMemory / 1024 / 1024).toFixed(2)} MB\n` +
        `Peak Memory: ${(peakMemory / 1024 / 1024).toFixed(2)} MB\n` +
        `Final Memory: ${(finalMemory / 1024 / 1024).toFixed(2)} MB\n` +
        `Memory Leak: ${(memoryLeak / 1024 / 1024).toFixed(2)} MB\n` +
        `Average Usage: ${(avgMemoryUsage / 1024 / 1024).toFixed(2)} MB\n` +
        `Memory Efficiency: ${memoryEfficiency}\n` +
        `Leak Severity: ${leakSeverity}\n` +
        `Memory per Pixel: ${((peakMemory - initialMemory) / (1920 * 1080)).toFixed(2)} bytes/pixel\n\n`;

    appendResult('memoryResults', result);

    button.disabled = false;
}

function calculateMemoryEfficiency(memoryUsed) {
    const expectedMemory = 1920 * 1080 * 4 * 3; // RGB + processing buffers
    const efficiency = (expectedMemory / memoryUsed) * 100;

    if (efficiency > 80) return 'EXCELLENT - Minimal overhead';
    if (efficiency > 60) return 'GOOD - Reasonable overhead';
    if (efficiency > 40) return 'FAIR - Moderate overhead';
    if (efficiency > 20) return 'POOR - High overhead';
    return 'CRITICAL - Excessive memory usage';
}

function getLeakSeverity(leak) {
    const leakMB = leak / 1024 / 1024;
    if (leakMB < 0.5) return 'NONE - No significant leak';
    if (leakMB < 2) return 'MINOR - Small leak detected';
    if (leakMB < 10) return 'MODERATE - Noticeable leak';
    if (leakMB < 50) return 'SEVERE - Significant leak';
    return 'CRITICAL - Major memory leak';
}

async function runAllComponentTests() {
    const components = ['colorspace', 'filtering', 'composite', 'noise', 'decode'];
    for (const component of components) {
        await runComponentTest(component);
    }
}

function collectAllData() {
    const dataArea = document.getElementById('dataCollection');
    dataArea.textContent = '';

    appendResult('dataCollection', '=== NTSC PERFORMANCE DATA COLLECTION ===\n\n');
    appendResult('dataCollection', `Timestamp: ${new Date().toISOString()}\n`);
    appendResult('dataCollection', `Browser: ${navigator.userAgent}\n`);
    appendResult('dataCollection', `Platform: ${navigator.platform}\n`);
    appendResult('dataCollection', `Memory: ${performance.memory ? (performance.memory.usedJSHeapSize / 1024 / 1024).toFixed(2) + ' MB' : 'N/A'}\n\n`);

    // Component Performance Data
    if (Object.keys(testResults.components).length > 0) {
        appendResult('dataCollection', '--- COMPONENT PERFORMANCE DATA ---\n');
        Object.entries(testResults.components).forEach(([component, data]) => {
            appendResult('dataCollection', `${component.toUpperCase()}:\n`);
            appendResult('dataCollection', `  Average Time: ${data.avgTime.toFixed(3)} ms\n`);
            appendResult('dataCollection', `  Min Time: ${data.minTime.toFixed(3)} ms\n`);
            appendResult('dataCollection', `  Max Time: ${data.maxTime.toFixed(3)} ms\n`);
            appendResult('dataCollection', `  Standard Deviation: ${data.stdDev.toFixed(3)} ms\n`);
            appendResult('dataCollection', `  Throughput: ${data.throughput.toFixed(2)} ops/sec\n`);
            appendResult('dataCollection', `  Iterations: ${data.iterations}\n\n`);
        });
    }

    // Resolution Scaling Data
    if (Object.keys(testResults.scaling).length > 0) {
        appendResult('dataCollection', '--- RESOLUTION SCALING DATA ---\n');
        Object.entries(testResults.scaling).forEach(([resolution, data]) => {
            appendResult('dataCollection', `${resolution}:\n`);
            appendResult('dataCollection', `  Average Time: ${data.avgTime.toFixed(3)} ms\n`);
            appendResult('dataCollection', `  Pixel Count: ${data.pixelCount.toLocaleString()}\n`);
            appendResult('dataCollection', `  Time per Pixel: ${data.timePerPixel.toFixed(3)} ns/pixel\n`);
            appendResult('dataCollection', `  Scaling Efficiency: ${data.scalingEfficiency}\n\n`);
        });
    }

    // Memory Usage Data
    if (testResults.memory && testResults.memory.initialMemory !== undefined) {
        appendResult('dataCollection', '--- MEMORY USAGE DATA ---\n');
        appendResult('dataCollection', `Initial Memory: ${(testResults.memory.initialMemory / 1024 / 1024).toFixed(3)} MB\n`);
        appendResult('dataCollection', `Peak Memory: ${(testResults.memory.peakMemory / 1024 / 1024).toFixed(3)} MB\n`);
        appendResult('dataCollection', `Final Memory: ${(testResults.memory.finalMemory / 1024 / 1024).toFixed(3)} MB\n`);
        appendResult('dataCollection', `Memory Leak: ${(testResults.memory.memoryLeak / 1024 / 1024).toFixed(3)} MB\n`);
        appendResult('dataCollection', `Average Usage: ${(testResults.memory.avgMemoryUsage / 1024 / 1024).toFixed(3)} MB\n`);
        appendResult('dataCollection', `Memory Efficiency: ${testResults.memory.memoryEfficiency}\n`);
        appendResult('dataCollection', `Leak Severity: ${testResults.memory.leakSeverity}\n\n`);
    }

    // System Performance Data
    appendResult('dataCollection', '--- SYSTEM PERFORMANCE DATA ---\n');
    appendResult('dataCollection', `CPU Cores: ${navigator.hardwareConcurrency || 'Unknown'}\n`);
    appendResult('dataCollection', `Connection Type: ${navigator.connection ? navigator.connection.effectiveType : 'Unknown'}\n`);
    appendResult('dataCollection', `Device Memory: ${navigator.deviceMemory ? navigator.deviceMemory + ' GB' : 'Unknown'}\n`);

    if (performance.timing) {
        const timing = performance.timing;
        appendResult('dataCollection', `Page Load Time: ${timing.loadEventEnd - timing.navigationStart} ms\n`);
        appendResult('dataCollection', `DOM Ready Time: ${timing.domContentLoadedEventEnd - timing.navigationStart} ms\n`);
    }

    appendResult('dataCollection', '\n--- DATA COLLECTION COMPLETE ---\n');
}

function clearResults(elementId) {
    const element = document.getElementById(elementId);
    if (element) {
        element.textContent = '';
    }
}

function exportResults() {
    const results = {
        timestamp: new Date().toISOString(),
        testResults: testResults,
        componentAnalysis: document.getElementById('componentResults').value,
        scalingAnalysis: document.getElementById('scalingResults').value,
        memoryAnalysis: document.getElementById('memoryResults').value,
        dataCollection: document.getElementById('dataCollection').value,
        summary: {
            totalComponents: Object.keys(testResults.components).length,
            slowestComponent: Object.keys(testResults.components).length > 0 ?
                Object.entries(testResults.components).sort(([, a], [, b]) => b.avgTime - a.avgTime)[0][0] : null,
            memoryLeakMB: testResults.memory.memoryLeak ?
                (testResults.memory.memoryLeak / 1024 / 1024).toFixed(2) : null
        }
    };

    const blob = new Blob([JSON.stringify(results, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `ntsc-bottleneck-analysis-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
}

window.onload = function () {
    initWorker();

    document.getElementById('fileInput').addEventListener('change', function (e) {
        const file = e.target.files[0];
        if (file && file.type.startsWith('image/')) {
            const reader = new FileReader();
            reader.onload = function (event) {
                currentImageData = event.target.result;
                const testImage = document.getElementById('testImage');
                testImage.src = currentImageData;
                testImage.style.display = 'block';
            };
            reader.readAsDataURL(file);
        }
    });

    generateTestImage(640, 480);
};