import React from 'react';
import { AbsoluteFill, OffthreadVideo, useCurrentFrame, useVideoConfig } from 'remotion';

// Caption type definition
export interface Caption {
  start: number;
  end: number;
  text: string;
}

// Component props
interface CaptionedVideoProps {
  videoUrl: string;
  captions: Caption[];
  style: 'bottom' | 'top-bar' | 'karaoke';
}

/**
 * Main Remotion composition that renders video with captions
 * Supports Hinglish (Devanagari + Latin) via NotoSans fonts
 */
export const CaptionedVideo: React.FC<CaptionedVideoProps> = ({
  videoUrl,
  captions,
  style
}) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const currentTime = frame / fps;

  // Debug logging on first frame
  if (frame === 0) {
    console.log('=== CAPTIONED VIDEO PROPS ===');
    console.log('videoUrl:', videoUrl);
    console.log('videoUrl type:', typeof videoUrl);
    console.log('videoUrl length:', videoUrl?.length);
    console.log('captions count:', captions?.length);
    console.log('style:', style);
    console.log('=== END PROPS ===');
  }

  // Fonts will be loaded via CSS @font-face instead of JS
  // This avoids React rendering issues

  // Find active caption for current time
  const activeCaption = captions.find(
    (caption) => currentTime >= caption.start && currentTime <= caption.end
  );

  return (
    <AbsoluteFill style={{ backgroundColor: 'black' }}>
      {/* Video layer - use OffthreadVideo for better performance and reliability */}
      <OffthreadVideo
        src={videoUrl}
        style={{ width: '100%', height: '100%', objectFit: 'contain' }}
      />

      {/* Caption overlay */}
      {activeCaption && (
        <AbsoluteFill>
          {style === 'bottom' && <BottomCaption text={activeCaption.text} />}
          {style === 'top-bar' && <TopBarCaption text={activeCaption.text} />}
          {style === 'karaoke' && (
            <KaraokeCaption
              text={activeCaption.text}
              progress={(currentTime - activeCaption.start) / (activeCaption.end - activeCaption.start)}
            />
          )}
        </AbsoluteFill>
      )}
    </AbsoluteFill>
  );
};

/**
 * Bottom-centered caption style
 * White text with black outline, positioned at bottom
 */
const BottomCaption: React.FC<{ text: string }> = ({ text }) => {
  // FIX #6: Strip HTML tags to prevent XSS
  const sanitizedText = text.replace(/<\/?[^>]+(>|$)/g, "");
  
  return (
    <div
      style={{
        position: 'absolute',
        bottom: 100,
        left: 0,
        right: 0,
        display: 'flex',
        justifyContent: 'center',
        padding: '0 40px'
      }}
    >
      <div
        style={{
          fontFamily: '"Noto Sans", "Noto Sans Devanagari", sans-serif',
          fontSize: 48,
          fontWeight: 'bold',
          color: 'white',
          textAlign: 'center',
          textShadow: `
            -2px -2px 0 #000,
            2px -2px 0 #000,
            -2px 2px 0 #000,
            2px 2px 0 #000,
            0 0 10px rgba(0,0,0,0.8)
          `,
          maxWidth: '80%',
          lineHeight: 1.4
        }}
      >
        {sanitizedText}
      </div>
    </div>
  );
};

/**
 * Top bar caption style
 * Black background bar with white text at top
 */
const TopBarCaption: React.FC<{ text: string }> = ({ text }) => {
  // FIX #6: Strip HTML tags to prevent XSS
  const sanitizedText = text.replace(/<\/?[^>]+(>|$)/g, "");
  
  return (
    <div
      style={{
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        backgroundColor: 'rgba(0, 0, 0, 0.85)',
        padding: '20px 40px',
        display: 'flex',
        justifyContent: 'center'
      }}
    >
      <div
        style={{
          fontFamily: '"Noto Sans", "Noto Sans Devanagari", sans-serif',
          fontSize: 42,
          fontWeight: '600',
          color: 'white',
          textAlign: 'center',
          maxWidth: '90%',
          lineHeight: 1.3
        }}
      >
        {sanitizedText}
      </div>
    </div>
  );
};

/**
 * Karaoke-style caption
 * Progressive highlight effect as words are spoken
 */
const KaraokeCaption: React.FC<{ text: string; progress: number }> = ({ text, progress }) => {
  // FIX #6: Strip HTML tags to prevent XSS
  const sanitizedText = text.replace(/<\/?[^>]+(>|$)/g, "");
  const words = sanitizedText.split(' ');
  const highlightedWordCount = Math.floor(words.length * progress);

  return (
    <div
      style={{
        position: 'absolute',
        bottom: 100,
        left: 0,
        right: 0,
        display: 'flex',
        justifyContent: 'center',
        padding: '0 40px'
      }}
    >
      <div
        style={{
          fontFamily: '"Noto Sans", "Noto Sans Devanagari", sans-serif',
          fontSize: 48,
          fontWeight: 'bold',
          textAlign: 'center',
          maxWidth: '80%',
          lineHeight: 1.4,
          display: 'flex',
          flexWrap: 'wrap',
          gap: '12px',
          justifyContent: 'center'
        }}
      >
        {words.map((word, index) => (
          <span
            key={index}
            style={{
              color: index < highlightedWordCount ? '#FFD700' : 'white',
              textShadow: `
                -2px -2px 0 #000,
                2px -2px 0 #000,
                -2px 2px 0 #000,
                2px 2px 0 #000,
                0 0 10px rgba(0,0,0,0.8)
              `,
              transition: 'color 0.1s ease'
            }}
          >
            {word}
          </span>
        ))}
      </div>
    </div>
  );
};
