let wasmWorker = null;
let wasmReady = false;
let currentImageData = null;
let currentRequestId = 0;
let processingRequestId = null;

// Initialize Web Worker
function initWorker() {
    wasmWorker = new Worker('worker.js');

    wasmWorker.onmessage = function (event) {
        const data = event.data;
        if (data.type === 'wasmLoaded') {
            wasmReady = true;
            document.getElementById('wasmStatus').textContent = 'Ready to process images!';

            // Initialize debug mode based on checkbox state
            const debugEnabled = document.getElementById('enableDebugLog').checked;
            wasmWorker.postMessage({ type: 'setDebugMode', enabled: debugEnabled });

            if (currentImageData) {
                processImage();
            }
        } else if (data.type === 'result') {
            if (data.requestId && data.requestId !== processingRequestId) {
                return;
            }
            const processedImage = document.getElementById('processedImage');
            processedImage.src = data.imageData;
            document.getElementById('processStatus').style.display = 'none';
            document.getElementById('processStatus').textContent = 'Processing time: ' + data.processTime + ' ms';
            document.getElementById('processStatus').style.display = 'block';
            processingRequestId = null;
        } else if (data.type === 'error') {
            if (data.requestId && data.requestId !== processingRequestId) {
                return;
            }
            showError(data.message);
            document.getElementById('processStatus').style.display = 'none';
            processingRequestId = null;

        }
    };

    wasmWorker.onerror = function (error) {
        console.error('Worker error:', error);
        showError('Web Worker encountered an error.');
        document.getElementById('wasmStatus').textContent = 'Failed to load WebAssembly module';
        document.getElementById('loading').style.display = 'none';

    };
}

initWorker(); // Call to initialize the worker

document.getElementById('fileInput').addEventListener('change', (e) => {
    if (e.target.files.length > 0) {
        handleFile(e.target.files[0]);
        document.getElementById('videoControls').style.display = 'none';
    }
});

document.getElementById('videoInput').addEventListener('change', function (event) {
    const file = event.target.files[0];
    if (file) {
        currentVideoFile = file;
        document.getElementById('videoControls').style.display = 'block';
        // Clear image data when video is selected
        currentImageData = null;
        document.getElementById('originalImage').src = '';
        document.getElementById('processedImage').src = '';
        document.getElementById('imageDisplay').style.display = 'none';
    }
});

// Add event listeners for video processing buttons
document.getElementById('processVideoBtn').addEventListener('click', processVideo);
document.getElementById('stopVideoBtn').addEventListener('click', stopVideoProcessing);

function handleFile(file) {
    if (!file.type.startsWith('image/')) {
        showError('Please select an image file');
        return;
    }

    const reader = new FileReader();
    reader.onload = (e) => {
        currentImageData = e.target.result;
        document.getElementById('originalImage').src = currentImageData;
        document.getElementById('imageDisplay').style.display = 'block';

        if (wasmReady) {
            processImage();
        }
    };
    reader.readAsDataURL(file);
}

function updateSliderValues() {
    const sliders = document.querySelectorAll('input[type="range"]');
    sliders.forEach(slider => {
        const valueElement = document.getElementById(slider.id + 'Value');
        if (valueElement) {
            valueElement.textContent = slider.value;
            slider.addEventListener('input', () => {
                valueElement.textContent = slider.value;
                if (currentImageData && wasmReady) {
                    processImage();
                }
            });
        }
    });

    const checkboxes = document.querySelectorAll('input[type="checkbox"]');
    checkboxes.forEach(checkbox => {
        if (checkbox.id === 'enableDebugLog') {
            checkbox.addEventListener('change', () => {
                if (wasmReady) {
                    wasmWorker.postMessage({ type: 'setDebugMode', enabled: checkbox.checked });
                }
            });
        } else {
            checkbox.addEventListener('change', () => {
                if (currentImageData && wasmReady) {
                    processImage();
                }
            });
        }
    });
}

function loadPreset(presetName) {
    if (!wasmReady) {
        showError('WebAssembly module not ready');
        return;
    }

    if (presetName === 'random') {
        loadRandomPreset();
        return;
    }

    // Send message to worker to get preset config
    wasmWorker.postMessage({ type: 'getPreset', presetName: presetName });

    wasmWorker.onmessage = function (event) {
        const data = event.data;
        if (data.type === 'presetConfig') {
            const config = JSON.parse(data.config);
            document.getElementById('compositePreemphasis').value = config.CompositePreemphasis || 0;
            document.getElementById('compositePreemphasisCut').value = config.CompositePreemphasisCut || 1000000;
            document.getElementById('compositeInChromaLowpass').checked = config.CompositeInChromaLowpass !== undefined ? config.CompositeInChromaLowpass : true;
            document.getElementById('compositeOutChromaLowpass').checked = config.CompositeOutChromaLowpass !== undefined ? config.CompositeOutChromaLowpass : true;
            document.getElementById('compositeOutChromaLowpassLite').checked = config.CompositeOutChromaLowpassLite !== undefined ? config.CompositeOutChromaLowpassLite : true;
            document.getElementById('colorBleedBefore').checked = config.ColorBleedBefore !== undefined ? config.ColorBleedBefore : true;
            document.getElementById('colorBleedHoriz').value = config.ColorBleedHoriz || 0;
            document.getElementById('colorBleedVert').value = config.ColorBleedVert || 0;
            document.getElementById('ringing').value = config.Ringing || 0;
            document.getElementById('enableRinging2').checked = config.EnableRinging2 || false;
            document.getElementById('ringingPower').value = config.RingingPower || 2;
            document.getElementById('ringingShift').value = config.RingingShift || 0;
            document.getElementById('freqNoiseSize').value = config.FreqNoiseSize || 0;
            document.getElementById('freqNoiseAmplitude').value = config.FreqNoiseAmplitude || 2;
            document.getElementById('videoNoise').value = config.VideoNoise || 0;
            document.getElementById('videoChromaNoise').value = config.VideoChromaNoise || 0;
            document.getElementById('videoChromaPhaseNoise').value = config.VideoChromaPhaseNoise || 0;
            document.getElementById('videoChromaLoss').value = config.VideoChromaLoss || 0;
            document.getElementById('noColorSubcarrier').checked = config.NoColorSubcarrier || false;
            document.getElementById('emulatingVHS').checked = config.EmulatingVHS || false;
            document.getElementById('vhsOutSharpen').value = config.VHSOutSharpen || 0;
            document.getElementById('vhsEdgeWave').value = config.VHSEdgeWave || 0;
            document.getElementById('vhsHeadSwitching').checked = config.VHSHeadSwitching || false;
            document.getElementById('vhsChromaVertBlend').checked = config.VHSChromaVertBlend || false;
            document.getElementById('vhsSVideoOut').checked = config.VHSSVideoOut || false;
            document.getElementById('outputVHSTapeSpeed').value = config.OutputVHSTapeSpeed || 0;
            document.getElementById('headSwitchingSpeed').value = config.HeadSwitchingSpeed || 0;
            document.getElementById('videoScanlinePhaseShift').value = config.VideoScanlinePhaseShift || 0;
            document.getElementById('videoScanlinePhaseShiftOffset').value = config.VideoScanlinePhaseShiftOffset || 0;
            document.getElementById('subcarrierAmplitude').value = config.SubcarrierAmplitude || 0;
            document.getElementById('outputNTSC').checked = config.OutputNTSC !== undefined ? config.OutputNTSC : true;
            document.getElementById('blackLineCut').checked = config.BlackLineCut || false;
            document.getElementById('precise').checked = config.Precise || false;
            document.getElementById('randomSeed').value = config.RandomSeed || 12345;
            document.getElementById('randomSeed2').value = config.RandomSeed2 || 67890;
            updateSliderValues();
        } else if (data.type === 'error') {
            showError(data.message);
        }
        // Re-attach the main onmessage handler after preset is loaded
        initWorker();
    };
}

async function processImage() {
    if (!wasmReady) {
        showError('WebAssembly module not ready');
        return;
    }

    if (!currentImageData) {
        showError('Please upload an image first');
        return;
    }

    // Generate new request ID and cancel any previous processing
    currentRequestId++;
    const requestId = currentRequestId;
    processingRequestId = requestId;

    const processStatus = document.getElementById('processStatus');
    const errorDiv = document.getElementById('error');
    processStatus.style.display = 'block';
    processStatus.textContent = 'Processing image...';
    errorDiv.style.display = 'none';

    const config = {
        CompositePreemphasis: parseFloat(document.getElementById('compositePreemphasis').value),
        CompositePreemphasisCut: parseFloat(document.getElementById('compositePreemphasisCut').value),
        ColorBleedBefore: document.getElementById('colorBleedBefore').checked,
        ColorBleedHoriz: parseInt(document.getElementById('colorBleedHoriz').value),
        ColorBleedVert: parseInt(document.getElementById('colorBleedVert').value),
        Ringing: parseFloat(document.getElementById('ringing').value),
        EnableRinging2: document.getElementById('enableRinging2').checked,
        RingingPower: parseInt(document.getElementById('ringingPower').value),
        RingingShift: parseInt(document.getElementById('ringingShift').value),
        FreqNoiseSize: parseFloat(document.getElementById('freqNoiseSize').value),
        FreqNoiseAmplitude: parseFloat(document.getElementById('freqNoiseAmplitude').value),
        CompositeInChromaLowpass: document.getElementById('compositeInChromaLowpass').checked,
        CompositeOutChromaLowpass: document.getElementById('compositeOutChromaLowpass').checked,
        CompositeOutChromaLowpassLite: document.getElementById('compositeOutChromaLowpassLite').checked,
        VideoNoise: parseInt(document.getElementById('videoNoise').value),
        VideoChromaNoise: parseInt(document.getElementById('videoChromaNoise').value),
        VideoChromaPhaseNoise: parseInt(document.getElementById('videoChromaPhaseNoise').value),
        VideoChromaLoss: parseInt(document.getElementById('videoChromaLoss').value),
        SubcarrierAmplitude: parseInt(document.getElementById('subcarrierAmplitude').value),
        SubcarrierAmplitudeBack: parseInt(document.getElementById('subcarrierAmplitude').value),
        EmulatingVHS: document.getElementById('emulatingVHS').checked,
        NoColorSubcarrier: document.getElementById('noColorSubcarrier').checked,
        VHSChromaVertBlend: document.getElementById('vhsChromaVertBlend').checked,
        VHSSVideoOut: document.getElementById('vhsSVideoOut').checked,
        VHSOutSharpen: parseFloat(document.getElementById('vhsOutSharpen').value),
        VHSEdgeWave: parseInt(document.getElementById('vhsEdgeWave').value),
        VHSHeadSwitching: document.getElementById('vhsHeadSwitching').checked,
        VHSHeadSwitchingPhaseNoise: 0.05,
        OutputNTSC: document.getElementById('outputNTSC').checked,
        VideoScanlinePhaseShift: parseInt(document.getElementById('videoScanlinePhaseShift').value),
        VideoScanlinePhaseShiftOffset: parseInt(document.getElementById('videoScanlinePhaseShiftOffset').value),
        OutputVHSTapeSpeed: parseInt(document.getElementById('outputVHSTapeSpeed').value),
        HeadSwitchingSpeed: parseInt(document.getElementById('headSwitchingSpeed').value),
        BlackLineCut: document.getElementById('blackLineCut').checked,
        Precise: document.getElementById('precise').checked,
        RandomSeed: parseInt(document.getElementById('randomSeed').value),
        RandomSeed2: parseInt(document.getElementById('randomSeed2').value)
    };

    const enableCompression = document.getElementById('enableCompression').checked;
    const maxWidth = enableCompression ? parseInt(document.getElementById('maxWidth').value) || 0 : 0;
    const maxHeight = enableCompression ? parseInt(document.getElementById('maxHeight').value) || 0 : 0;

    let imageDataToSend = currentImageData;

    if (enableCompression && (maxWidth > 0 || maxHeight > 0)) {
        try {
            imageDataToSend = await resizeImage(currentImageData, maxWidth, maxHeight);
        } catch (error) {
            if (requestId === currentRequestId) {
                showError('Failed to resize image: ' + error.message);
                processStatus.style.display = 'none';
            }
            return;
        }
    }

    // Check if this request is still current before sending
    if (requestId !== currentRequestId) {
        return;
    }

    const request = {
        imageData: imageDataToSend,
        config: config,
        maxWidth: maxWidth,
        maxHeight: maxHeight,
        requestId: requestId
    };

    // Send message to worker to process image
    wasmWorker.postMessage({ type: 'processImage', request: request });
}

async function resizeImage(base64Data, maxWidth, maxHeight) {
    return new Promise((resolve, reject) => {
        const img = new Image();
        img.onload = () => {
            const canvas = document.createElement('canvas');
            let width = img.width;
            let height = img.height;

            if (maxWidth > 0 && width > maxWidth) {
                height = height * (maxWidth / width);
                width = maxWidth;
            }

            if (maxHeight > 0 && height > maxHeight) {
                width = width * (maxHeight / height);
                height = maxHeight;
            }

            canvas.width = width;
            canvas.height = height;

            const ctx = canvas.getContext('2d');
            ctx.drawImage(img, 0, 0, width, height);

            resolve(canvas.toDataURL('image/jpeg', 0.9)); // Use JPEG for better performance
        };
        img.onerror = (error) => {
            reject(error);
        };
        img.src = base64Data;
    });
}

function showError(message) {
    const errorDiv = document.getElementById('error');
    errorDiv.textContent = message;
    errorDiv.style.display = 'block';
}

function loadRandomPreset() {
    // Generate random values for all parameters
    document.getElementById('compositePreemphasis').value = (Math.random() * 8).toFixed(1);
    document.getElementById('compositePreemphasisCut').value = Math.floor(Math.random() * 1900000) + 100000;
    document.getElementById('compositeInChromaLowpass').checked = Math.random() > 0.5;
    document.getElementById('compositeOutChromaLowpass').checked = Math.random() > 0.5;
    document.getElementById('compositeOutChromaLowpassLite').checked = Math.random() > 0.5;
    document.getElementById('colorBleedBefore').checked = Math.random() > 0.5;
    document.getElementById('colorBleedHoriz').value = Math.floor(Math.random() * 21);
    document.getElementById('colorBleedVert').value = Math.floor(Math.random() * 21);
    document.getElementById('ringing').value = (Math.random() * 3).toFixed(1);
    document.getElementById('enableRinging2').checked = Math.random() > 0.5;
    document.getElementById('ringingPower').value = Math.floor(Math.random() * 5) + 1;
    document.getElementById('ringingShift').value = Math.floor(Math.random() * 21) - 10;
    document.getElementById('freqNoiseSize').value = (Math.random() * 10).toFixed(1);
    document.getElementById('freqNoiseAmplitude').value = (Math.random() * 10).toFixed(1);
    document.getElementById('videoNoise').value = Math.floor(Math.random() * 101);
    document.getElementById('videoChromaNoise').value = Math.floor(Math.random() * 501);
    document.getElementById('videoChromaPhaseNoise').value = Math.floor(Math.random() * 101);
    document.getElementById('videoChromaLoss').value = Math.floor(Math.random() * 100001 / 100) * 100;
    document.getElementById('noColorSubcarrier').checked = Math.random() > 0.8;
    document.getElementById('emulatingVHS').checked = Math.random() > 0.5;
    document.getElementById('vhsOutSharpen').value = (Math.random() * 5).toFixed(1);
    document.getElementById('vhsEdgeWave').value = Math.floor(Math.random() * 11);
    document.getElementById('vhsHeadSwitching').checked = Math.random() > 0.7;
    document.getElementById('vhsChromaVertBlend').checked = Math.random() > 0.3;
    document.getElementById('vhsSVideoOut').checked = Math.random() > 0.7;
    document.getElementById('outputVHSTapeSpeed').value = Math.floor(Math.random() * 3);
    document.getElementById('headSwitchingSpeed').value = Math.floor(Math.random() * 11);
    document.getElementById('videoScanlinePhaseShift').value = [0, 90, 180, 270][Math.floor(Math.random() * 4)];
    document.getElementById('videoScanlinePhaseShiftOffset').value = Math.floor(Math.random() * 4);
    document.getElementById('subcarrierAmplitude').value = Math.floor(Math.random() * 101);
    document.getElementById('outputNTSC').checked = Math.random() > 0.2;
    document.getElementById('blackLineCut').checked = Math.random() > 0.7;
    document.getElementById('precise').checked = Math.random() > 0.5;
    document.getElementById('randomSeed').value = Math.floor(Math.random() * 4294967295);
    document.getElementById('randomSeed2').value = Math.floor(Math.random() * 4294967295);

    updateSliderValues();

    if (currentImageData) {
        processImage();
    }
}

let currentVideoFile = null;
let videoProcessing = false;
let processedFrames = [];
let originalFrames = [];
let totalFrames = 0;
let currentFrameIndex = 0;

async function processVideo() {
    if (!wasmReady) {
        showError('WebAssembly module not ready');
        return;
    }

    if (!currentVideoFile) {
        showError('Please select a video file first');
        return;
    }

    videoProcessing = true;
    processedFrames = [];
    originalFrames = [];
    currentFrameIndex = 0;

    const processBtn = document.getElementById('processVideoBtn');
    const stopBtn = document.getElementById('stopVideoBtn');
    const progressDiv = document.getElementById('videoProgress');
    const progressBar = document.getElementById('videoProgressBar');
    const progressText = document.getElementById('videoProgressText');

    processBtn.style.display = 'none';
    stopBtn.style.display = 'inline-block';
    progressDiv.style.display = 'block';
    progressBar.value = 0;
    progressText.textContent = '0%';

    let video = null;
    let canvas = null;

    try {
        video = document.createElement('video');
        video.src = URL.createObjectURL(currentVideoFile);
        video.muted = true;

        await new Promise((resolve, reject) => {
            video.onloadedmetadata = resolve;
            video.onerror = reject;
        });

        const frameRate = parseInt(document.getElementById('videoFrameRate').value) || 30;
        const duration = video.duration;
        totalFrames = Math.floor(duration * frameRate);
        progressBar.max = totalFrames;

        canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');

        const maxWidth = parseInt(document.getElementById('videoMaxWidth').value) || 0;
        const maxHeight = parseInt(document.getElementById('videoMaxHeight').value) || 0;

        // Set canvas size based on video dimensions and max constraints
        let canvasWidth = video.videoWidth;
        let canvasHeight = video.videoHeight;

        if (maxWidth > 0 && canvasWidth > maxWidth) {
            canvasHeight = canvasHeight * (maxWidth / canvasWidth);
            canvasWidth = maxWidth;
        }

        if (maxHeight > 0 && canvasHeight > maxHeight) {
            canvasWidth = canvasWidth * (maxHeight / canvasHeight);
            canvasHeight = maxHeight;
        }

        canvas.width = canvasWidth;
        canvas.height = canvasHeight;

        const config = getCurrentConfig();

        const batchSize = 8;
        for (let batchStart = 0; batchStart < totalFrames && videoProcessing; batchStart += batchSize) {
            const batchEnd = Math.min(batchStart + batchSize, totalFrames);
            const batchPromises = [];

            for (let frame = batchStart; frame < batchEnd && videoProcessing; frame++) {
                const time = frame / frameRate;
                video.currentTime = time;

                await new Promise(resolve => {
                    video.onseeked = resolve;
                });

                ctx.drawImage(video, 0, 0, canvasWidth, canvasHeight);
                const frameData = canvas.toDataURL('image/jpeg', 0.9);

                originalFrames.push(frameData);
                batchPromises.push(processVideoFrame(frameData, config, frame, totalFrames, time));
            }

            const batchResults = await Promise.all(batchPromises);
            processedFrames.push(...batchResults);

            currentFrameIndex = batchEnd;
            const progress = Math.round((currentFrameIndex / totalFrames) * 100);
            progressBar.value = currentFrameIndex;
            progressText.textContent = `${progress}% (${currentFrameIndex}/${totalFrames})`;
        }

        if (videoProcessing) {
            showVideoPreview();
        }

    } catch (error) {
        showError('Video processing failed: ' + error.message);
    } finally {
        videoProcessing = false;
        processBtn.style.display = 'inline-block';
        stopBtn.style.display = 'none';
        if (!videoProcessing) {
            progressDiv.style.display = 'none';
        }

        // Clean up resources
        if (video && video.src) {
            URL.revokeObjectURL(video.src);
        }
        if (canvas) {
            canvas.width = 0;
            canvas.height = 0;
        }
    }
}

async function processVideoFrame(frameData, config, frameNumber, totalFrames, timestamp) {
    return new Promise((resolve, reject) => {
        const requestId = Date.now() + Math.random();

        const tempHandler = (event) => {
            const data = event.data;
            if (data.requestId === requestId) {
                wasmWorker.removeEventListener('message', tempHandler);
                if (data.type === 'videoFrameResult') {
                    resolve(data.imageData);
                } else if (data.type === 'error') {
                    reject(new Error(data.message));
                }
            }
        };

        wasmWorker.addEventListener('message', tempHandler);

        const request = {
            imageData: frameData,
            config: config,
            frameNumber: frameNumber,
            totalFrames: totalFrames,
            timestamp: timestamp,
            requestId: requestId
        };

        wasmWorker.postMessage({ type: 'processVideoFrame', request: request });
    });
}

function getCurrentConfig() {
    return {
        CompositePreemphasis: parseFloat(document.getElementById('compositePreemphasis').value),
        CompositePreemphasisCut: parseFloat(document.getElementById('compositePreemphasisCut').value),
        ColorBleedBefore: document.getElementById('colorBleedBefore').checked,
        ColorBleedHoriz: parseInt(document.getElementById('colorBleedHoriz').value),
        ColorBleedVert: parseInt(document.getElementById('colorBleedVert').value),
        Ringing: parseFloat(document.getElementById('ringing').value),
        EnableRinging2: document.getElementById('enableRinging2').checked,
        RingingPower: parseInt(document.getElementById('ringingPower').value),
        RingingShift: parseInt(document.getElementById('ringingShift').value),
        FreqNoiseSize: parseFloat(document.getElementById('freqNoiseSize').value),
        FreqNoiseAmplitude: parseFloat(document.getElementById('freqNoiseAmplitude').value),
        CompositeInChromaLowpass: document.getElementById('compositeInChromaLowpass').checked,
        CompositeOutChromaLowpass: document.getElementById('compositeOutChromaLowpass').checked,
        CompositeOutChromaLowpassLite: document.getElementById('compositeOutChromaLowpassLite').checked,
        VideoNoise: parseInt(document.getElementById('videoNoise').value),
        VideoChromaNoise: parseInt(document.getElementById('videoChromaNoise').value),
        VideoChromaPhaseNoise: parseInt(document.getElementById('videoChromaPhaseNoise').value),
        VideoChromaLoss: parseInt(document.getElementById('videoChromaLoss').value),
        SubcarrierAmplitude: parseInt(document.getElementById('subcarrierAmplitude').value),
        SubcarrierAmplitudeBack: parseInt(document.getElementById('subcarrierAmplitude').value),
        EmulatingVHS: document.getElementById('emulatingVHS').checked,
        NoColorSubcarrier: document.getElementById('noColorSubcarrier').checked,
        VHSChromaVertBlend: document.getElementById('vhsChromaVertBlend').checked,
        VHSSVideoOut: document.getElementById('vhsSVideoOut').checked,
        VHSOutSharpen: parseFloat(document.getElementById('vhsOutSharpen').value),
        VHSEdgeWave: parseInt(document.getElementById('vhsEdgeWave').value),
        VHSHeadSwitching: document.getElementById('vhsHeadSwitching').checked,
        VHSHeadSwitchingPhaseNoise: 0.05,
        OutputNTSC: document.getElementById('outputNTSC').checked,
        VideoScanlinePhaseShift: parseInt(document.getElementById('videoScanlinePhaseShift').value),
        VideoScanlinePhaseShiftOffset: parseInt(document.getElementById('videoScanlinePhaseShiftOffset').value),
        OutputVHSTapeSpeed: parseInt(document.getElementById('outputVHSTapeSpeed').value),
        HeadSwitchingSpeed: parseInt(document.getElementById('headSwitchingSpeed').value),
        BlackLineCut: document.getElementById('blackLineCut').checked,
        Precise: document.getElementById('precise').checked,
        RandomSeed: parseInt(document.getElementById('randomSeed').value),
        RandomSeed2: parseInt(document.getElementById('randomSeed2').value)
    };
}

function stopVideoProcessing() {
    videoProcessing = false;
    document.getElementById('processVideoBtn').style.display = 'inline-block';
    document.getElementById('stopVideoBtn').style.display = 'none';
    document.getElementById('videoProgress').style.display = 'none';
}

let currentPreviewFrame = 0;
let isPlaying = false;
let playInterval = null;

function showVideoPreview() {
    if (processedFrames.length === 0 || originalFrames.length === 0) {
        showError('No processed frames available');
        return;
    }

    document.getElementById('videoDisplay').style.display = 'block';
    document.getElementById('imageDisplay').style.display = 'none';

    createVideoFromFrames();
}

async function createVideoFromFrames() {
    if (processedFrames.length === 0 || originalFrames.length === 0) return;

    try {
        const frameRate = parseInt(document.getElementById('videoFrameRate').value) || 30;

        // Show loading indicator in console
        console.log('Creating video previews...');

        // Create both videos in parallel for faster processing
        const [originalVideoBlob, processedVideoBlob] = await Promise.all([
            createVideoBlob(originalFrames, frameRate),
            createVideoBlob(processedFrames, frameRate)
        ]);

        // Set video sources
        const originalVideoUrl = URL.createObjectURL(originalVideoBlob);
        const originalVideo = document.getElementById('originalVideo');
        originalVideo.src = originalVideoUrl;

        const processedVideoUrl = URL.createObjectURL(processedVideoBlob);
        const processedVideo = document.getElementById('processedVideo');
        processedVideo.src = processedVideoUrl;

        console.log(`Videos created: ${processedFrames.length} frames at ${frameRate} FPS`);

    } catch (error) {
        showError('Failed to create videos: ' + error.message);
    }
}

// Optimized batch preload for faster processing
async function preloadImages(frames) {
    const images = [];
    const batchSize = 10;

    for (let i = 0; i < frames.length; i += batchSize) {
        const batch = frames.slice(i, i + batchSize);
        const batchPromises = batch.map((frameSrc, batchIndex) => {
            const actualIndex = i + batchIndex;
            return new Promise((resolve) => {
                const img = new Image();
                img.onload = () => {
                    images[actualIndex] = img;
                    resolve();
                };
                img.src = frameSrc;
            });
        });

        await Promise.all(batchPromises);
    }

    return images;
}

async function createVideoBlob(frames, frameRate) {
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');

    // Preload all images first for faster processing
    const images = await preloadImages(frames);

    // Get dimensions from first image
    canvas.width = images[0].width;
    canvas.height = images[0].height;

    const stream = canvas.captureStream(frameRate);
    const mediaRecorder = new MediaRecorder(stream, {
        mimeType: 'video/webm;codecs=vp8',
        videoBitsPerSecond: 8000000
    });

    const chunks = [];
    mediaRecorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
            chunks.push(event.data);
        }
    };

    return new Promise((resolve, reject) => {
        mediaRecorder.onstop = () => {
            const blob = new Blob(chunks, { type: 'video/webm' });
            resolve(blob);
        };

        mediaRecorder.onerror = reject;

        mediaRecorder.start();

        // Optimized frame drawing using preloaded images
        let frameIndex = 0;
        const frameDuration = Math.max(8, 1000 / frameRate / 2); // Faster frame processing

        const drawFrame = () => {
            if (frameIndex < images.length) {
                // Draw preloaded image directly - much faster
                ctx.drawImage(images[frameIndex], 0, 0);
                frameIndex++;

                if (frameIndex < images.length) {
                    setTimeout(drawFrame, frameDuration);
                } else {
                    // Add a small delay before stopping to ensure last frame is captured
                    setTimeout(() => mediaRecorder.stop(), 50);
                }
            }
        };

        drawFrame();
    });
}



function downloadProcessedFrames() {
    if (processedFrames.length === 0) {
        showError('No processed frames to download');
        return;
    }

    processedFrames.forEach((frameData, index) => {
        const link = document.createElement('a');
        link.download = `processed_frame_${String(index + 1).padStart(4, '0')}.png`;
        link.href = frameData;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
    });


}

async function exportVideo() {
    if (processedFrames.length === 0) {
        showError('No processed frames to export');
        return;
    }

    try {
        const exportBtn = document.getElementById('exportVideoBtn');
        exportBtn.disabled = true;
        exportBtn.textContent = 'Exporting...';

        const frameRate = parseInt(document.getElementById('videoFrameRate').value) || 30;
        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');

        // Get dimensions from first frame
        const firstImg = new Image();
        await new Promise((resolve) => {
            firstImg.onload = resolve;
            firstImg.src = processedFrames[0];
        });

        canvas.width = firstImg.width;
        canvas.height = firstImg.height;

        const stream = canvas.captureStream(frameRate);
        const mediaRecorder = new MediaRecorder(stream, {
            mimeType: 'video/webm;codecs=vp8',
            videoBitsPerSecond: 2500000
        });

        const chunks = [];
        mediaRecorder.ondataavailable = (event) => {
            if (event.data.size > 0) {
                chunks.push(event.data);
            }
        };

        mediaRecorder.onstop = () => {
            const blob = new Blob(chunks, { type: 'video/webm' });
            const url = URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.href = url;
            link.download = 'ntsc_processed_video.webm';
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
            URL.revokeObjectURL(url);

            exportBtn.disabled = false;
            exportBtn.textContent = 'Export Video';
        };

        mediaRecorder.start();

        // Draw frames to canvas at specified frame rate
        for (let i = 0; i < processedFrames.length; i++) {
            const img = new Image();
            await new Promise((resolve) => {
                img.onload = () => {
                    ctx.drawImage(img, 0, 0);
                    resolve();
                };
                img.src = processedFrames[i];
            });

            // Wait for frame duration
            await new Promise(resolve => setTimeout(resolve, 5));
        }

        mediaRecorder.stop();

    } catch (error) {
        console.error('Video export failed:', error);
        showError('Video export failed: ' + error.message);

        const exportBtn = document.getElementById('exportVideoBtn');
        exportBtn.disabled = false;
        exportBtn.textContent = 'Export Video';
    }
}

function showSuccess(message) {
    const errorDiv = document.getElementById('error');
    errorDiv.textContent = message;
    errorDiv.style.display = 'block';
    errorDiv.style.color = 'green';
    setTimeout(() => {
        errorDiv.style.color = '';
        errorDiv.style.display = 'none';
    }, 3000);
}

function toggleCompressionSettings() {
    const enableCompression = document.getElementById('enableCompression');
    const compressionSettings = document.getElementById('compressionSettings');

    compressionSettings.style.display = enableCompression.checked ? 'block' : 'none';
}

document.getElementById('enableCompression').addEventListener('change', toggleCompressionSettings);

updateSliderValues();
toggleCompressionSettings();

// Add real-time listeners for number inputs
const numberInputs = document.querySelectorAll('input[type="number"]');
numberInputs.forEach(input => {
    input.addEventListener('input', () => {
        if (currentImageData && wasmReady) {
            processImage();
        }
    });
});

// Add real-time listener for compression checkbox
document.getElementById('enableCompression').addEventListener('change', () => {
    if (currentImageData && wasmReady) {
        processImage();
    }
});

document.addEventListener("DOMContentLoaded", function () {
    renderMathInElement(document.body, {
        delimiters: [
            { left: '\\(', right: '\\)', display: false },
            { left: '\\[', right: '\\]', display: true }
        ]
    });
});