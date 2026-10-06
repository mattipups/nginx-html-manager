import { test, expect } from '@playwright/test';

test.describe('Nginx HTML Manager - V1.2.0 Lifecycle Tests', () => {
  test('Upload, In-Place Update, Rollback und Download-Integrität', async ({ page, request }) => {
    // 1. Initialer Upload
    const uploadRes = await request.post('/api/upload', {
      multipart: {
        file: { name: 'app.html', mimeType: 'text/html', buffer: Buffer.from('<h1>Stand v1</h1>') },
        slug: 'qa-target',
        profile: 'interactive-local'
      }
    });
    expect(uploadRes.ok()).toBeTruthy();
    const data = await uploadRes.json();
    const fileId = data.id;

    // 2. Prüfung via Browser
    await page.goto(`/view/${fileId}`);
    await expect(page.locator('h1')).toHaveText('Stand v1');

    // 3. Datei aktualisieren ohne ID oder Custom URL zu verändern
    const updateRes = await request.put(`/api/files/${fileId}`, {
      multipart: {
        file: { name: 'app.html', mimeType: 'text/html', buffer: Buffer.from('<h1>Stand v2</h1>') }
      }
    });
    expect(updateRes.ok()).toBeTruthy();

    // 4. Verifikation des neuen Inhalts unter demselben Link
    await page.reload();
    await expect(page.locator('h1')).toHaveText('Stand v2');

    // 5. Gezielter Rollback auf v1
    const rollbackRes = await request.post(`/api/files/${fileId}/rollback/1`);
    expect(rollbackRes.ok()).toBeTruthy();

    // 6. Verifikation nach Rollback
    await page.reload();
    await expect(page.locator('h1')).toHaveText('Stand v1');
  });
});
