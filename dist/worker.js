// web/worker.js
importScripts('wasm_exec.js');

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
    const { type, request, presetName, enabled } = e.data;

    if (type === 'setDebugMode') {
        try {
            setDebugMode(enabled);
        } catch (error) {
            console.error('Failed to set debug mode:', error);
        }
    } else if (type === 'processImage') {
        try {
            let imageData, config, requestId;
            
            if (e.data.request) {
                ({ imageData, config, requestId } = e.data.request);
            } else {
                imageData = e.data.imageData;
                config = e.data.config || {};
                requestId = e.data.requestId;
            }
            
            const startTime = performance.now();
            const result = processNTSC(JSON.stringify({ imageData, config }));
            const endTime = performance.now();
            const processTime = (endTime - startTime).toFixed(1);

            if (result.error) {
                postMessage({ 
                    type: 'error', 
                    message: result.error,
                    requestId: requestId 
                });
            } else {
                postMessage({ 
                    type: 'result', 
                    imageData: result.imageData, 
                    processTime: processTime,
                    requestId: requestId 
                });
            }
        } catch (error) {
            postMessage({ 
                type: 'error', 
                message: 'Processing failed in worker: ' + error.message,
                requestId: e.data.requestId || (e.data.request ? e.data.request.requestId : null) 
            });
        }
    } else if (type === 'processVideoFrame') {
        try {
            let imageData, config, requestId, frameNumber, totalFrames, timestamp;
            
            if (e.data.request) {
                ({ imageData, config, requestId, frameNumber, totalFrames, timestamp } = e.data.request);
            } else {
                imageData = e.data.imageData;
                config = e.data.config || {};
                requestId = e.data.requestId;
                frameNumber = e.data.frameNumber || 0;
                totalFrames = e.data.totalFrames;
                timestamp = e.data.timestamp;
            }
            
            const startTime = performance.now();
            const result = processVideoFrame(JSON.stringify({ 
                imageData, 
                config, 
                frameNumber, 
                totalFrames, 
                timestamp 
            }));
            const endTime = performance.now();
            const processTime = (endTime - startTime).toFixed(1);

            if (result.error) {
                postMessage({ 
                    type: 'error', 
                    message: result.error,
                    requestId: requestId,
                    frameNumber: frameNumber
                });
            } else {
                postMessage({ 
                    type: 'videoFrameResult', 
                    imageData: result.imageData, 
                    processTime: processTime,
                    requestId: requestId,
                    frameNumber: result.frameNumber || frameNumber
                });
            }
        } catch (error) {
            postMessage({ 
                type: 'error', 
                message: 'Video frame processing failed in worker: ' + error.message,
                requestId: e.data.requestId || (e.data.request ? e.data.request.requestId : null),
                frameNumber: e.data.frameNumber || (e.data.request ? e.data.request.frameNumber : null)
            });
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
