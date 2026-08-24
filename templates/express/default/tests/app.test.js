import { describe, expect, it } from 'vitest'
import request from 'supertest'
import app from '../src/app.js'

describe('API', () => {
  it('GET /health returns ok', async () => {
    const res = await request(app).get('/health')
    expect(res.status).toBe(200)
    expect(res.body.status).toBe('ok')
  })

  it('POST /api/sum adds two numbers', async () => {
    const res = await request(app).post('/api/sum').send({ a: 2, b: 3 })
    expect(res.status).toBe(200)
    expect(res.body.result).toBe(5)
  })

  it('POST /api/sum rejects invalid input', async () => {
    const res = await request(app).post('/api/sum').send({ a: 'x', b: 3 })
    expect(res.status).toBe(400)
  })
})
