# Technical Implementation

The technical foundation of this simulation is a comprehensive digital signal processing pipeline implemented in Go, meticulously engineered to emulate the complex artifacts and signal degradations inherent in the NTSC analog video standard. The system operates at the fundamental sampling rate of 14.318 MHz, corresponding to four times the NTSC color subcarrier frequency of 3.579545 MHz, ensuring accurate temporal and spectral representation of the analog signal characteristics.

## Color Space Transformation and Quantization

The process initiates with the transformation of the input image's color representation from the conventional RGB model to the YIQ color space, a critical step that decouples luminance (Y) from chrominance (I and Q) components. This separation is fundamental to NTSC encoding, as it enables backward compatibility with monochrome receivers while facilitating color transmission. The forward transformation from RGB to YIQ is defined by the standardized matrix operation:

$$
\begin{bmatrix} Y \\ I \\ Q \end{bmatrix} =
\begin{bmatrix} 0.299 & 0.587 & 0.114 \\ 0.596 & -0.274 & -0.322 \\ 0.211 & -0.523 & 0.312 \end{bmatrix}
\begin{bmatrix} R \\ G \\ B \end{bmatrix}
$$

In the implementation, the Y component is computed first using the weighted sum $Y = 0.299R + 0.587G + 0.114B$, followed by the chrominance components calculated as $I = -0.27(B-Y) + 0.74(R-Y)$ and $Q = 0.41(B-Y) + 0.48(R-Y)$. To maintain numerical precision and facilitate subsequent integer arithmetic operations, the YIQ components are scaled by a factor of 256 and stored as fixed-point integers. The inverse transformation for display purposes employs the matrix:

$$
\begin{bmatrix} R \\ G \\ B \end{bmatrix} =
\begin{bmatrix} 1.000 & 0.956 & 0.621 \\ 1.000 & -0.272 & -0.647 \\ 1.000 & -1.106 & 1.703 \end{bmatrix}
\begin{bmatrix} Y \\ I \\ Q \end{bmatrix}
$$

## Analog Bandwidth Simulation and Filtering

The analog transmission medium imposes severe bandwidth constraints on the video signal, necessitating sophisticated filtering to simulate these limitations accurately. The system employs a first-order Infinite Impulse Response (IIR) low-pass filter, characterized by the recursive difference equation:

$$ y[n] = \alpha \cdot y[n-1] + (1-\alpha) \cdot x[n] $$

where the filter coefficient $\alpha$ is derived from the desired cutoff frequency $f_c$ and sampling rate $f_s$ through the relationship $\alpha = e^{-2\pi f_c / f_s}$. The NTSC standard specifies distinct bandwidth allocations for luminance and chrominance: the Y channel extends to approximately 4.2 MHz, while the I and Q channels are limited to 1.3 MHz and 0.4 MHz respectively. These asymmetric bandwidth constraints are implemented through separate filter instances with appropriately configured cutoff frequencies.

## Composite Pre-emphasis and De-emphasis

To compensate for the high-frequency attenuation inherent in analog transmission and enhance perceived image sharpness, the NTSC system incorporates pre-emphasis and de-emphasis mechanisms. The pre-emphasis stage selectively amplifies higher frequencies in the luminance channel through a high-pass filtering operation combined with gain control. The mathematical formulation is expressed as:

$$ Y_{out}[n] = Y_{in}[n] + \epsilon \cdot (Y_{in}[n] - \alpha_{hp} \cdot Y_{in}[n-1]) $$

where $\epsilon$ represents the pre-emphasis coefficient controlling the magnitude of high-frequency boost, and $\alpha_{hp}$ defines the high-pass filter characteristics. This process effectively creates a "peaking" response that counteracts subsequent low-pass filtering effects while introducing the characteristic "sharpening" artifacts observed in analog video systems.

## Chrominance Subcarrier Modulation and Demodulation Artifacts

In the NTSC composite signal, chrominance information is quadrature-modulated onto a 3.579545 MHz color subcarrier using the I and Q components as in-phase and quadrature signals respectively. The modulated chrominance signal is mathematically represented as:

$$ C(t) = I(t) \cos(2\pi f_{sc} t) + Q(t) \sin(2\pi f_{sc} t) $$

where $f_{sc}$ denotes the subcarrier frequency. Demodulation imperfections in analog receivers often result in spatial misalignment between luminance and chrominance components, manifesting as the characteristic "color bleeding" artifact. This phenomenon is simulated by applying independent spatial offsets to the I and Q components:

$$ I_{out}(x, y) = I_{in}(x - \Delta x_I, y - \Delta y_I) $$
$$ Q_{out}(x, y) = Q_{in}(x - \Delta x_Q, y - \Delta y_Q) $$

where $(\Delta x_I, \Delta y_I)$ and $(\Delta x_Q, \Delta y_Q)$ represent user-configurable displacement vectors that simulate timing errors and phase distortions in the demodulation process.

## Frequency Domain Ringing and Gibbs Phenomena

Sharp transitions in the video signal, when subjected to bandwidth-limited transmission, exhibit characteristic "ringing" artifacts due to the Gibbs phenomenon. This effect is particularly pronounced at high-contrast edges where the finite bandwidth of the transmission medium cannot adequately represent the sharp discontinuities. The simulation implements this through frequency-domain processing using the Discrete Fourier Transform:

$$ X[k] = \sum_{n=0}^{N-1} x[n] e^{-j2\pi kn/N} $$

A custom band-pass filter $H[k]$ is applied in the frequency domain to selectively attenuate or emphasize specific frequency components, particularly those around the color subcarrier frequency. The filter response is designed to emulate the characteristics of analog notch filters used in NTSC decoders:

$$ H[k] = \begin{cases}
1 & \text{if } |k - k_{sc}| > \Delta k \\
\beta & \text{if } |k - k_{sc}| \leq \Delta k
\end{cases} $$

where $k_{sc}$ corresponds to the subcarrier frequency bin, $\Delta k$ defines the notch width, and $\beta$ represents the attenuation factor. The processed signal is then transformed back to the time domain using the Inverse Discrete Fourier Transform.

## Stochastic Noise Generation and Temporal Correlation

Analog video signals are inherently susceptible to various noise sources, including thermal noise, electromagnetic interference, and quantization artifacts. The system implements a sophisticated noise generation module based on the XorWow pseudo-random number generator, which provides excellent statistical properties and computational efficiency. The generator maintains internal state variables $(x, y, z, w, v, d)$ and produces pseudo-random sequences through the recurrence relation:

$$ \begin{align}
t &= x \oplus (x \gg 2) \\
x &= y, \quad y = z, \quad z = w, \quad w = v \\
v &= (v \oplus (v \ll 4)) \oplus (t \oplus (t \ll 1)) \\
d &= d + 362437
\end{align} $$

The system provides two distinct noise processing modes to accommodate different computational requirements and accuracy demands. The non-precise mode employs a computationally efficient approach where a large pre-computed noise array is smoothed using a simple exponential moving average filter with $\alpha = 0.5$. In contrast, the precise mode generates noise dynamically for each scanline, implementing temporal correlation through an accumulative smoothing process:

$$ n[i] = \frac{n[i-1] + r[i]}{2} + \frac{n[i-1] + r[i-1]}{4} $$

where $n[i]$ represents the correlated noise value at pixel $i$, and $r[i]$ denotes the raw random number. This formulation introduces both spatial and temporal correlation, more accurately reflecting the characteristics of real analog noise.

## VHS Tape Degradation Modeling

The simulation incorporates detailed modeling of VHS tape-specific degradations to enhance authenticity. The mechanical instabilities inherent in VHS playback systems manifest as time-base errors, simulated through the "edge wave" effect. This is implemented by generating random horizontal displacement values $\delta[j]$ for each scanline $j$, followed by low-pass filtering to create smooth, correlated variations:

$$ \delta_{filtered}[j] = \alpha_{wave} \cdot \delta_{filtered}[j-1] + (1-\alpha_{wave}) \cdot \delta[j] $$

The head-switching noise effect replicates the signal disruption that occurs during the transition between video heads in the helical scan mechanism. This is modeled as a localized distortion band, typically positioned in the lower portion of the frame, where both geometric displacement and additive noise are applied with increased intensity. The mathematical representation involves a spatial weighting function $w(y)$ that defines the affected region:

$$ w(y) = \begin{cases}
0 & \text{if } y < y_{start} \\
\sin^2\left(\frac{\pi(y - y_{start})}{y_{end} - y_{start}}\right) & \text{if } y_{start} \leq y \leq y_{end} \\
0 & \text{if } y > y_{end}
\end{cases} $$

Chrominance degradation, a common artifact in aged VHS tapes, is simulated through controlled attenuation of the I and Q components using a degradation factor $\gamma$:

$$ I_{degraded} = \gamma \cdot I_{original}, \quad Q_{degraded} = \gamma \cdot Q_{original} $$

where $\gamma$ ranges from 0 (complete chroma loss) to 1 (no degradation), allowing for precise control over the degree of color desaturation.

## Interlaced Field Processing and Temporal Artifacts

The NTSC standard employs interlaced scanning, where each frame consists of two fields containing alternating scanlines. This temporal sampling introduces specific artifacts that must be accurately modeled. The system processes odd and even fields independently, maintaining separate processing contexts to preserve the temporal characteristics of the interlaced format. Field-specific artifacts, such as inter-field motion blur and temporal aliasing, are simulated through controlled blending of adjacent field data:

$$ F_{blended}[i] = \alpha_{temporal} \cdot F_{current}[i] + (1-\alpha_{temporal}) \cdot F_{previous}[i] $$

where $F_{current}$ and $F_{previous}$ represent the current and previous field data respectively, and $\alpha_{temporal}$ controls the degree of temporal blending.

## Advanced Chroma Modulation and Demodulation Pipeline

The implementation features a sophisticated chroma processing pipeline that accurately models the NTSC composite signal generation and decoding process. The `chromaIntoLuma` function simulates the encoding stage where I and Q chrominance components are modulated onto the luminance signal using quadrature amplitude modulation. The modulation process employs pre-computed lookup tables for the subcarrier phase relationships:

$$ U_{mult} = [1, 0, -1, 0], \quad V_{mult} = [0, 1, 0, -1] $$

The composite signal generation is mathematically expressed as:

$$ Y_{composite}[x] = Y_{original}[x] + \frac{A_{sc}}{50} \cdot (I[x] \cdot U_{mult}[\xi + x \bmod 4] + Q[x] \cdot V_{mult}[\xi + x \bmod 4]) $$

where $A_{sc}$ represents the subcarrier amplitude and $\xi$ denotes the phase offset determined by the scanline position and field number. The demodulation process, implemented in `chromaFromLuma`, employs a sophisticated comb filter approach to separate luminance and chrominance components. The algorithm utilizes a four-point moving average filter to extract the luminance component:

$$ Y_{separated}[x] = \frac{1}{4} \sum_{i=0}^{3} Y_{composite}[x-2+i] $$

The extracted chrominance signal undergoes phase-sensitive demodulation with alternating sign correction to recover the original I and Q components, followed by interpolation to restore full bandwidth.

## Memory Management and Performance Optimization

The implementation incorporates advanced memory management strategies to achieve optimal performance in resource-constrained environments. Pre-allocated buffer pools are utilized to minimize garbage collection overhead during intensive processing operations. The `ChromaBuffers` structure maintains persistent memory allocations for intermediate processing arrays:

$$ \text{BufferSet} = \{chroma, y_2, y_{d4}, sums, sums_0, acc, acc_4, cxi, cxi_1\} $$

This approach eliminates repeated memory allocation and deallocation cycles, resulting in significant performance improvements for real-time processing applications. The system employs fixed-point arithmetic throughout the processing pipeline, utilizing 32-bit signed integers to represent fractional values with implicit scaling factors. This design choice provides computational efficiency while maintaining sufficient numerical precision for high-quality output.

## Fast Fourier Transform Implementation

The frequency-domain processing capabilities are powered by a custom radix-2 Cooley-Tukey FFT implementation optimized for power-of-two input sizes. The algorithm employs bit-reversal permutation for efficient in-place computation:

$$ X[k] = \sum_{n=0}^{N-1} x[n] \cdot W_N^{kn}, \quad W_N = e^{-j2\pi/N} $$

The implementation features specialized `fftShift` and `ifftShift` functions to facilitate zero-frequency centering for filter design applications. The frequency-domain ringing simulation applies custom transfer functions to emulate the characteristics of analog video equipment:

$$ H_{ring}[k] = \begin{cases}
1 & \text{if } |k - N/2| > \alpha \cdot N/2 \\
\beta + \eta \cdot \mathcal{N}(0,1) & \text{otherwise}
\end{cases} $$

where $\alpha$ controls the filter bandwidth, $\beta$ represents the attenuation factor, and $\eta \cdot \mathcal{N}(0,1)$ introduces controlled frequency-domain noise to simulate analog component tolerances.

## Adaptive VHS Tape Speed Modeling

The VHS emulation subsystem incorporates detailed modeling of different tape speeds (SP, LP, EP) with corresponding bandwidth limitations and temporal artifacts. Each speed mode is characterized by specific luminance and chrominance cutoff frequencies:

$$ \begin{align}
\text{SP Mode:} & \quad f_{Y,cut} = 2.4\text{ MHz}, \quad f_{C,cut} = 320\text{ kHz} \\
\text{LP Mode:} & \quad f_{Y,cut} = 1.9\text{ MHz}, \quad f_{C,cut} = 300\text{ kHz} \\
\text{EP Mode:} & \quad f_{Y,cut} = 1.4\text{ MHz}, \quad f_{C,cut} = 280\text{ kHz}
\end{align} $$

The chrominance delay compensation varies with tape speed to simulate the mechanical characteristics of different recording densities. The VHS sharpening algorithm applies controlled high-frequency emphasis to compensate for magnetic tape frequency response limitations:

$$ Y_{sharpened}[x] = Y_{original}[x] + \gamma_{sharpen} \cdot (Y_{original}[x] - Y_{lowpass}[x]) $$

where $\gamma_{sharpen}$ represents the sharpening coefficient and $Y_{lowpass}$ denotes the low-pass filtered luminance signal.

## Scanline Phase Relationship and Color Subcarrier Synchronization

The NTSC color subcarrier maintains a precise phase relationship with the horizontal sync signal to ensure proper color reproduction. The implementation accurately models this relationship through the `chromaLumaXi` function, which calculates the subcarrier phase offset for each scanline:

$$ \xi(field, y) = \begin{cases}
(field + offset + \lfloor y/2 \rfloor) \bmod 4 & \text{if } \phi = 90° \\
((field + y) \bmod 2 + offset) \bmod 4 & \text{if } \phi = 180° \\
(field + offset) \bmod 4 & \text{if } \phi = 270° \\
offset \bmod 4 & \text{otherwise}
\end{cases} $$

where $\phi$ represents the configured phase shift, $field$ denotes the current field number, and $offset$ provides additional phase adjustment. This precise phase control enables accurate simulation of color artifacts such as rainbow effects and dot crawl patterns that result from subcarrier timing errors in analog systems.

This comprehensive signal processing pipeline, operating at the authentic NTSC sampling rate and incorporating mathematically rigorous models of analog video artifacts, successfully reproduces the complex visual characteristics of vintage television and VHS playback systems with exceptional fidelity and technical accuracy. The modular architecture facilitates precise control over individual artifact components while maintaining computational efficiency suitable for real-time applications.