import { Composition, registerRoot } from 'remotion';
import { CaptionedVideo } from './CaptionedVideo';

export const RemotionRoot: React.FC = () => {
  return (
    <>
      <Composition
        id="CaptionedVideo"
        component={CaptionedVideo}
        durationInFrames={1800}
        fps={30}
        width={1920}
        height={1080}
        defaultProps={{
          videoUrl: '',
          captions: [],
          style: 'bottom'
        }}
        calculateMetadata={async ({ props }) => {
          // Use the last caption's end time to determine duration
          if (props.captions && props.captions.length > 0) {
            const lastCaption = props.captions[props.captions.length - 1];
            const durationInSeconds = Math.ceil(lastCaption.end) + 1;
            const durationInFrames = Math.ceil(durationInSeconds * 30);
            return {
              durationInFrames,
              fps: 30,
              width: 1920,
              height: 1080,
            };
          }
          return {
            durationInFrames: 1800,
            fps: 30,
            width: 1920,
            height: 1080,
          };
        }}
      />
    </>
  );
};

// Register the root component
registerRoot(RemotionRoot);
