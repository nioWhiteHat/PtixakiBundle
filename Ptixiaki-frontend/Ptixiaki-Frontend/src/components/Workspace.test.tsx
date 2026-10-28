import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import Workspace from './Workspace'

describe('workspace navigation', () => {
  it('starts on Browse with the menu closed', async () => {
    const user = userEvent.setup()
    render(<Workspace />)
    expect(screen.getByRole('heading', { name: 'Browse properties.' })).toBeVisible()
    expect(screen.queryByRole('navigation', { name: 'Main navigation' })).not.toBeInTheDocument()
    const toggle = screen.getByRole('button', { name: 'Open navigation' })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    await user.click(toggle)
    expect(toggle).toHaveAttribute('aria-expanded', 'true')
    const navigation = screen.getByRole('navigation', { name: 'Main navigation' })
    expect(within(navigation).getAllByRole('link').map((button) => button.textContent)).toEqual(['Browse', 'Portfolio', 'Analysis'])
    expect(within(navigation).getByRole('link', { name: 'Browse' })).toHaveAttribute('aria-current', 'page')
    await user.click(toggle)
    expect(screen.queryByRole('navigation', { name: 'Main navigation' })).not.toBeInTheDocument()
  })

  it('switches between Analysis search and Browse without running analysis', async () => {
    const user = userEvent.setup()
    const fetchMock = vi.fn<typeof fetch>().mockImplementation(async () => {
      return Response.json({ index: 1, page_size: 20, has_next: false, properties: [] })
    })
    vi.stubGlobal('fetch', fetchMock)
    render(<Workspace />)
    for (const section of ['Analysis', 'Browse']) {
      await user.click(screen.getByRole('button', { name: 'Open navigation' }))
      await user.click(screen.getByRole('link', { name: section }))
      const main = await screen.findByRole('main', { name: section })
      if (section === 'Browse') {
        expect(within(main).getByRole('heading', { name: 'Browse properties.' })).toBeVisible()
      } else {
        expect(within(main).getByRole('combobox', { name: 'Area name' })).toBeVisible()
      }
      expect(document.title).toBe(`${section} · Property Co Pilot`)
      expect(screen.getByRole('main', { name: section })).toHaveFocus()
      await user.click(screen.getByRole('button', { name: 'Open navigation' }))
      expect(screen.getByRole('link', { name: section })).toHaveAttribute('aria-current', 'page')
      await user.click(screen.getByRole('button', { name: 'Close navigation' }))
    }
    expect(fetchMock).toHaveBeenCalled()
    expect(fetchMock.mock.calls.every((call) => String(call[0]).startsWith('/User/BrowseProperties'))).toBe(true)
  })

  it('supports keyboard opening, Escape dismissal and restoring focus', async () => {
    const user = userEvent.setup()
    render(<Workspace />)
    await user.keyboard('{Enter}')
    await user.tab()
    expect(screen.getByRole('link', { name: 'Browse' })).toHaveFocus()
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('navigation', { name: 'Main navigation' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Open navigation' })).toHaveFocus()
  })

  it('closes when clicking outside the sidebar', async () => {
    const user = userEvent.setup()
    render(<Workspace />)
    await user.click(screen.getByRole('button', { name: 'Open navigation' }))
    await user.click(screen.getByRole('button', { name: 'Dismiss navigation' }))
    expect(screen.queryByRole('navigation', { name: 'Main navigation' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Open navigation' })).toHaveFocus()
  })
})
