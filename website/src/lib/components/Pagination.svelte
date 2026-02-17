<script lang="ts">
  interface Props {
    current: number;
    total: number;
    onPage: (page: number) => void;
  }

  let { current, total, onPage }: Props = $props();

  // Simple range generator
  let pages = $derived(Array.from({ length: total }, (_, i) => i + 1));
  let visiblePages = $derived(
    pages.filter(
      (p) => p === 1 || p === total || (p >= current - 2 && p <= current + 2),
    ),
  );
</script>

<div class="d-none d-sm-block">
  <nav aria-label="Pages">
    <ul class="pagination">
      <li class="page-item {current === 1 ? 'disabled' : ''}">
        <button class="page-link" onclick={() => onPage(current - 1)}>«</button>
      </li>
      {#each pages as page}
        {#if page === 1 || page === total || (page >= current - 4 && page <= current + 4)}
          <li class="page-item {current === page ? 'active' : ''}">
            <button class="page-link" onclick={() => onPage(page)}
              >{page}</button
            >
          </li>
        {:else if page === current - 5 || page === current + 5}
          <li class="page-item disabled"><span class="page-link">...</span></li>
        {/if}
      {/each}
      <li class="page-item {current === total ? 'disabled' : ''}">
        <button class="page-link" onclick={() => onPage(current + 1)}>»</button>
      </li>
    </ul>
  </nav>
</div>

<div class="d-sm-none">
  <nav aria-label="Pages">
    <ul class="pagination pagination-sm">
      <li class="page-item {current === 1 ? 'disabled' : ''}">
        <button class="page-link" onclick={() => onPage(current - 1)}>«</button>
      </li>
      {#each pages as page}
        {#if page === 1 || page === total || (page >= current - 1 && page <= current + 1)}
          <li class="page-item {current === page ? 'active' : ''}">
            <button class="page-link" onclick={() => onPage(page)}
              >{page}</button
            >
          </li>
        {:else if page === current - 2 || page === current + 2}
          <li class="page-item disabled"><span class="page-link">...</span></li>
        {/if}
      {/each}
      <li class="page-item {current === total ? 'disabled' : ''}">
        <button class="page-link" onclick={() => onPage(current + 1)}>»</button>
      </li>
    </ul>
  </nav>
</div>
