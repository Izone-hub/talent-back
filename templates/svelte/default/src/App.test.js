import { describe, expect, it } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/svelte'
import App from '../src/App.svelte'

describe('App', () => {
  it('increments the counter', async () => {
    render(App)
    expect(screen.getByTestId('count').textContent).toBe('Count: 0')
    await fireEvent.click(screen.getByTestId('inc'))
    expect(screen.getByTestId('count').textContent).toBe('Count: 1')
  })
})
