import { test, expect, type Page } from '@playwright/test';
async function login(page: Page, name: string) {
  await page.goto('/');
  await page.getByRole('button', { name: new RegExp(name) }).click();
  await expect(page.locator('#menu-toggle')).toBeAttached();
}
async function switchUser(page: Page, name: string) {
  await page.getByRole('button', { name: 'Сменить пользователя' }).click();
  await page.getByRole('button', { name: new RegExp(name) }).click();
}
async function ask(page: Page, message: string) {
  await page.getByRole('textbox', { name: 'Сообщение', exact: true }).fill(message);
  await page.getByRole('button', { name: 'Отправить', exact: true }).click();
  await expect(page.getByRole('status').filter({ hasText: 'Помощник обрабатывает' })).toHaveCount(
    0
  );
}
test('real API: absence approval and history persist across profiles and reload', async ({
  page,
}) => {
  await login(page, 'Анна');
  await ask(page, 'Оформи отпуск на 2030-01-10 по 2030-01-11');
  const card = page.getByTestId(/^action-/).last();
  await expect(card.getByRole('button', { name: 'Подтвердить отправку' })).toBeVisible();
  await card.getByRole('button', { name: 'Подтвердить отправку' }).click();
  await expect(card.getByText('Выполнено', { exact: true })).toBeVisible();
  await page.reload();
  await expect(page.getByRole('log')).toContainText('Оформи отпуск');
  await switchUser(page, 'Мария');
  await expect(
    page.getByRole('heading', { name: 'Решения, которые двигают вперёд.' })
  ).toBeVisible();
  await page.getByLabel('Комментарий сотруднику · необязательно').fill('Согласовано в E2E');
  await page.getByRole('button', { name: 'Одобрить заявку' }).click();
  await expect(page.getByText('Входящих заявок по выбранным условиям нет.')).toBeVisible();
  await switchUser(page, 'Анна');
  await page.getByRole('link', { name: 'Мои заявки', exact: true }).click();
  await page.getByRole('tab', { name: 'Одобрены', exact: true }).click();
  await expect(page.getByText('Согласовано в E2E')).toBeVisible();
  await page.reload();
  await page.getByRole('tab', { name: 'Одобрены', exact: true }).click();
  await expect(page.getByText('Согласовано в E2E')).toBeVisible();
  await switchUser(page, 'Борис');
  await page.getByRole('link', { name: 'Мои заявки', exact: true }).click();
  await expect(page.getByText('Согласовано в E2E')).toHaveCount(0);
});
test('real API: checklist and mentor confirmation', async ({ page }) => {
  await login(page, 'Борис');
  await page.getByRole('link', { name: 'План адаптации', exact: true }).click();
  const self = page.getByRole('combobox', {
    name: 'Статус: Подключиться к чатам и каналам команды',
  });
  if (await self.count()) await self.selectOption('done');
  const step = page.getByRole('combobox', {
    name: 'Статус: Обсудить с наставником цели и план адаптации',
  });
  await step.selectOption('waiting');
  await expect(page.getByText('Ожидает подтверждения наставником')).toBeVisible();
  await switchUser(page, 'Михаил');
  await page.getByRole('button').filter({ hasText: 'Борис Стажёр' }).click();
  await page.getByRole('button', { name: 'Подтвердить шаг', exact: true }).click();
  await expect(
    page.getByText('Нет шагов, ожидающих проверки по выбранным условиям.')
  ).toBeVisible();
  await switchUser(page, 'Борис');
  await page.getByRole('link', { name: 'План адаптации', exact: true }).click();
  const row = page
    .getByRole('article')
    .filter({ hasText: 'Обсудить с наставником цели и план адаптации' });
  await expect(row.getByText('Выполнено', { exact: true })).toBeVisible();
});
test('real API: cancel access action creates no request', async ({ page }) => {
  await login(page, 'Анна');
  await ask(page, 'Нужен доступ к billing-e2e на запись');
  const card = page.getByTestId(/^action-/).last();
  await card.getByRole('button', { name: 'Отменить', exact: true }).click();
  await expect(card.getByText('Отменено', { exact: true })).toBeVisible();
  await page.getByRole('link', { name: 'Мои заявки', exact: true }).click();
  await expect(page.getByRole('tabpanel')).not.toContainText('billing-e2e');
  await page.getByRole('tab', { name: 'Действия', exact: true }).click();
  await page.getByRole('tab', { name: 'Отменённые', exact: true }).click();
  await expect(page.getByRole('tabpanel')).toContainText('billing-e2e');
});
for (const width of [360, 390, 430, 820, 1024, 1366, 1440, 1920])
  test(`responsive ${width}: pages, mobile navigation, keyboard focus`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await login(page, 'Анна');
    for (const hash of ['chat', 'plan', 'requests', 'materials']) {
      await page.goto('/#' + hash);
      await expect(page.getByRole('main')).toBeVisible();
      await expect(page.getByText('Загрузка…')).toHaveCount(0);
      expect(
        await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)
      ).toBe(true);
      await page.screenshot({ path: `test-results/screens/${width}-${hash}.png`, fullPage: true });
    }
    if (width <= 800) {
      await page.getByRole('button', { name: 'Открыть меню' }).click();
      await expect(page.getByRole('link', { name: 'План адаптации', exact: true })).toBeVisible();
      await page.keyboard.press('Escape');
      await expect(page.getByRole('button', { name: 'Открыть меню' })).toBeFocused();
    }
    await page.goto('/');
    await page.keyboard.press('Tab');
    await expect(page.getByRole('link', { name: 'К содержимому' })).toBeFocused();
  });
test('real API: manager and HR metrics use actual periods', async ({ page }) => {
  await login(page, 'Ольга');
  await expect(page.getByRole('heading', { name: 'Адаптация в фокусе.' })).toBeVisible();
  for (const n of ['7', '90']) {
    const response = page.waitForResponse(r => r.url().includes('/api/metrics?days=' + n));
    await page.getByLabel('Период метрик').selectOption(n);
    await response;
    await expect(page.getByText('Загрузка…')).toHaveCount(0);
  }
  await page.goto('/#approvals');
  await expect(
    page.getByText('Раздел недоступен для вашей роли. Выберите раздел в меню.')
  ).toBeVisible();
});

test('real API: rejection and mentor return are reflected for intern', async ({ page }) => {
  await login(page, 'Анна');
  await ask(page, 'Оформи больничный на 2030-02-10');
  await page
    .getByTestId(/^action-/)
    .last()
    .getByRole('button', { name: 'Подтвердить отправку' })
    .click();
  await expect(
    page
      .getByTestId(/^action-/)
      .last()
      .getByText('Выполнено', { exact: true })
  ).toBeVisible();
  await switchUser(page, 'Мария');
  await page.getByLabel('Комментарий сотруднику · необязательно').fill('Уточните даты');
  await page.getByRole('button', { name: 'Отклонить', exact: true }).click();
  await expect(page.getByText('Входящих заявок по выбранным условиям нет.')).toBeVisible();
  await switchUser(page, 'Анна');
  await page.getByRole('link', { name: 'Мои заявки', exact: true }).click();
  await page.getByRole('tab', { name: 'Отклонены', exact: true }).click();
  await expect(page.getByText('Уточните даты')).toBeVisible();
  await page.getByRole('link', { name: 'План адаптации', exact: true }).click();
  await page
    .getByRole('combobox', { name: 'Статус: Согласовать промежуточные результаты с наставником' })
    .selectOption('waiting');
  await expect(page.getByText('Ожидает подтверждения наставником')).toBeVisible();
  await switchUser(page, 'Михаил');
  await page.getByRole('button', { name: 'Вернуть в работу' }).click();
  await expect(
    page.getByText('Нет шагов, ожидающих проверки по выбранным условиям.')
  ).toBeVisible();
  await switchUser(page, 'Анна');
  await page.getByRole('link', { name: 'План адаптации', exact: true }).click();
  await expect(
    page.getByRole('combobox', {
      name: 'Статус: Согласовать промежуточные результаты с наставником',
    })
  ).toHaveValue('in_progress');
});
for (const width of [360, 390, 430, 820, 1024, 1366, 1440, 1920])
  test(`responsive roles ${width}: queues and metrics`, async ({ page, request }) => {
    const intern = { Authorization: 'Bearer dev-intern-1' };
    await request.post('/api/progress/steps/cccccccc-0000-0000-0000-000000000007/status', {
      headers: intern,
      data: { status: 'waiting' },
    });
    const chat = await request.post('/api/chat', {
      headers: intern,
      data: { message: 'Нужен доступ к viewport-resource на администрирование' },
    });
    const result = await chat.json();
    await request.post(`/api/actions/${result.pending_actions[0].id}/confirm`, {
      headers: intern,
      data: {},
    });
    await page.setViewportSize({ width, height: 900 });
    for (const [name, hash] of [
      ['Михаил', 'mentor'],
      ['Мария', 'approvals'],
      ['Ольга', 'metrics'],
    ]) {
      if (await page.locator('#sidebar').count()) {
        if (width <= 800) await page.getByRole('button', { name: 'Открыть меню' }).click();
        await switchUser(page, name);
      } else await login(page, name);
      await expect(
        page.getByRole('heading', {
          name:
            hash === 'mentor'
              ? 'Помочь сделать следующий шаг.'
              : hash === 'approvals'
                ? 'Решения, которые двигают вперёд.'
                : 'Адаптация в фокусе.',
        })
      ).toBeVisible();
      if (hash === 'mentor')
        await expect(
          page.getByRole('button', { name: 'Подтвердить шаг', exact: true })
        ).toBeVisible();
      if (hash === 'approvals')
        await expect(
          page.getByRole('button', { name: 'Одобрить заявку', exact: true })
        ).toBeAttached();
      await expect(page.getByText('Загрузка…')).toHaveCount(0);
      await page.evaluate(() => document.fonts.ready);
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(
        true
      );
      await page.screenshot({ path: `test-results/screens/${width}-${hash}.png`, fullPage: true });
    }
  });
