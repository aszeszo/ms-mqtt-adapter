import { css } from "lit";

export const haTheme = css`
  :root {
    /* Spacing (4px base unit) */
    --ha-space-1: 4px;
    --ha-space-2: 8px;
    --ha-space-3: 12px;
    --ha-space-4: 16px;
    --ha-space-6: 24px;
    --ha-space-8: 32px;

    /* Border radius */
    --ha-border-radius-sm: 4px;
    --ha-border-radius-md: 8px;
    --ha-border-radius-lg: 12px;
    --ha-border-radius-xl: 16px;
    --ha-border-radius-pill: 9999px;

    /* Typography */
    --ha-font-family-body: Roboto, Noto, sans-serif;
    --ha-font-family-code: monospace;
    --ha-font-size-s: 12px;
    --ha-font-size-m: 14px;
    --ha-font-size-l: 16px;
    --ha-font-size-xl: 20px;
    --ha-font-size-2xl: 24px;
    --ha-font-weight-normal: 400;
    --ha-font-weight-medium: 500;
    --ha-font-weight-bold: 700;
    --ha-line-height-condensed: 1.2;
    --ha-line-height-normal: 1.6;

    /* Semantic colors - light mode */
    --primary-color: #03a9f4;
    --primary-text-color: #212121;
    --secondary-text-color: #727272;
    --disabled-text-color: #bdbdbd;
    --divider-color: #e0e0e0;
    --card-background-color: #ffffff;
    --primary-background-color: #fafafa;
    --error-color: #db4437;
    --warning-color: #ff9800;
    --success-color: #4caf50;
    --info-color: #03a9f4;

    /* Card variables */
    --ha-card-background: var(--card-background-color);
    --ha-card-border-radius: var(--ha-border-radius-lg);
    --ha-card-border-color: var(--divider-color);
    --ha-card-border-width: 1px;
    --ha-card-box-shadow: none;

    /* Animations */
    --ha-animation-duration-fast: 150ms;
    --ha-animation-duration-normal: 250ms;
  }

  @media (prefers-color-scheme: dark) {
    :root {
      --primary-text-color: #e1e1e1;
      --secondary-text-color: #9b9b9b;
      --disabled-text-color: #6f6f6f;
      --divider-color: #3a3a3a;
      --card-background-color: #1c1c1c;
      --primary-background-color: #111111;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    :root {
      --ha-animation-duration-fast: 0ms;
      --ha-animation-duration-normal: 0ms;
    }
  }
`;
