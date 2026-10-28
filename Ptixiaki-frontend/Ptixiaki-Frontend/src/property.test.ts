import { describe, expect, it, vi } from 'vitest'
import { getPropertyData, money, number, signed } from './property'
import { portfolioFixture } from './test/propertyFixtures'

describe('property API and formatting', () => {
  it('makes authenticated same-origin GET requests and returns the Go JSON contract', async () => {
    const fetchMock = vi.fn().mockResolvedValue(Response.json(portfolioFixture))
    vi.stubGlobal('fetch', fetchMock)
    const signal = new AbortController().signal
    expect(await getPropertyData('/User/Portfolio', signal)).toEqual(portfolioFixture)
    expect(fetchMock).toHaveBeenCalledWith('/User/Portfolio', expect.objectContaining({ credentials: 'include', headers: { Accept: 'application/json' }, signal: expect.any(AbortSignal) }))
  })

  it.each([401, 404, 500])('handles HTTP %i without leaking internal error text', async (status) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('secret SQL error', { status })))
    try {
      await getPropertyData('/User/Portfolio', new AbortController().signal)
      expect.fail('Expected an error')
    } catch (error) {
      expect(error).toHaveProperty('status', status)
      expect((error as Error).message).not.toContain('SQL')
    }
  })

  it('handles connection and malformed JSON failures', async () => {
    const fetchMock = vi.fn().mockRejectedValueOnce(new TypeError('Network failure')).mockResolvedValueOnce(new Response('<html>'))
    vi.stubGlobal('fetch', fetchMock)
    for (let i = 0; i < 2; i++) {
      await expect(getPropertyData('/User/Portfolio', new AbortController().signal)).rejects.toThrow('Couldn’t load property data')
    }
  })

  it('keeps null distinct from zero, preserves rental cents and signs', () => {
    expect(money(null)).toBe('—')
    expect(number(0)).toBe('0')
    expect(money(12.34)).toBe('€12.34')
    expect(signed(-2.5)).toBe('-2.5%')
    expect(signed(0)).toBe('0%')
  })
})
