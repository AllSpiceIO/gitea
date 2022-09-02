// @ts-check
import {test, expect} from '@playwright/test';
import {load_logged_in_context, login_user} from './utils_e2e.js';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

test('Create Repo', async ({browser}, workerInfo) => {
  const context = await load_logged_in_context(browser, workerInfo, 'user2');
  const page = await context.newPage();

  const response = await page.goto('/repo/create');
  await expect(response?.status()).toBe(200); // Status OK

  await page.type('input[name=repo_name]', `test-repo-${workerInfo.workerIndex}`);
  await page.click('form button.ui.green.button:visible');

  await expect(page.url()).toBe(`${workerInfo.project.use.baseURL}/user2/test-repo-${workerInfo.workerIndex}`);
});
