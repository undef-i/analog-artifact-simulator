// web/worker.js
importScripts('./wasm_exec.js');

const go = new Go();
let wasm;

async function loadWasm() {
    try {
        const result = await WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject);
        wasm = result.instance;
        go.run(wasm);
        postMessage({ type: 'wasmLoaded' });
    } catch (err) {
        console.error('Failed to load WASM in worker:', err);
        postMessage({ type: 'error', message: 'Failed to load WebAssembly module in worker.' });
    }
}

loadWasm();

onmessage = function(e) {
    const { type, request, presetName } = e.data;

    if (type === 'processImage') {
        try {
            const startTime = performance.now();
            const result = processNTSC(JSON.stringify(request));
            const endTime = performance.now();
            const processTime = (endTime - startTime).toFixed(1);

            if (result.error) {
                postMessage({ type: 'error', message: result.error });
            } else {
                postMessage({ type: 'result', imageData: result.imageData, processTime: processTime });
            }
        } catch (error) {
            postMessage({ type: 'error', message: 'Processing failed in worker: ' + error.message });
        }
    } else if (type === 'getPreset') {
        try {
            const result = getPreset(presetName);
            if (result.error) {
                postMessage({ type: 'error', message: result.error });
            } else {
                postMessage({ type: 'presetConfig', config: result.config });
            }
        } catch (error) {
            postMessage({ type: 'error', message: 'Failed to get preset in worker: ' + error.message });
        }
    }
};
