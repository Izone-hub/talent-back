<script>
  let count = 0
</script>

<main>
  <h1>Hello, Sandbox!</h1>
  <p data-testid="count">Count: {count}</p>
  <button data-testid="inc" on:click={() => count++}>Increment</button>
</main>
