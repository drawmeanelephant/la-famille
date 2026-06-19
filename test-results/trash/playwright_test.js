const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch();
  const context = await browser.newContext({
    recordVideo: {
      dir: 'test-results/',
    }
  });
  const page = await context.newPage();

  try {
    // Navigate to the docs page
    await page.goto('http://localhost:8000/docs/index.html');

    // Wait for a bit to ensure styles are applied
    await page.waitForTimeout(1000);

    // Take a full-page screenshot
    await page.screenshot({ path: 'test-results/sidebar-screenshot.png', fullPage: true });
    console.log('Screenshot taken successfully');
  } catch (error) {
    console.error('Error during test:', error);
  } finally {
    // Close context to save video
    await context.close();
    await browser.close();

    // Rename the video file to sidebar-video.webm
    const fs = require('fs');
    const path = require('path');
    const files = fs.readdirSync('test-results');
    const videoFile = files.find(f => f.endsWith('.webm') && f !== 'sidebar-video.webm');

    if (videoFile) {
      fs.renameSync(
        path.join('test-results', videoFile),
        path.join('test-results', 'sidebar-video.webm')
      );
      console.log('Video saved successfully');
    }
  }
})();
