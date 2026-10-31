let audioContext: AudioContext | null = null;

type WebkitAudioWindow = Window & typeof globalThis & { webkitAudioContext?: typeof AudioContext };

export const appIsUnfocused = () => document.visibilityState === "hidden" || !document.hasFocus();

export async function prepareDownloadCompletionSound() {
  const AudioContextClass = window.AudioContext || (window as WebkitAudioWindow).webkitAudioContext;
  if (!AudioContextClass) return;
  audioContext ||= new AudioContextClass();
  if (audioContext.state === "suspended") await audioContext.resume();
}

export async function playDownloadCompletionSound() {
  try {
    await prepareDownloadCompletionSound();
    if (!audioContext) return;
    const now = audioContext.currentTime;
    const oscillator = audioContext.createOscillator();
    const gain = audioContext.createGain();
    oscillator.type = "sine";
    oscillator.frequency.setValueAtTime(659.25, now);
    oscillator.frequency.setValueAtTime(880, now + 0.12);
    gain.gain.setValueAtTime(0.0001, now);
    gain.gain.exponentialRampToValueAtTime(0.12, now + 0.02);
    gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.32);
    oscillator.connect(gain);
    gain.connect(audioContext.destination);
    oscillator.start(now);
    oscillator.stop(now + 0.34);
  } catch {
    // Webview audio can remain blocked until the first user interaction.
  }
}
