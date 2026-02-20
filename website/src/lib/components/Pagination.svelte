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

<style lang="scss">
  .pagination {
    border-radius: var(--border-radius);
    overflow: hidden;
    display: flex;
    padding-left: 0;
    list-style: none;

    .page-item {
      .page-link {
        border-radius: 0;
        border: 1px solid var(--gray-300);
        margin: 0;
        color: var(--gray-700);
        background: #fff;
        display: flex;
        align-items: center;
        justify-content: center;
        min-width: 2.5rem;
        height: 2.5rem;
        padding: 0 0.75rem;
        font-size: 0.9rem;
        transition: all 0.2s ease;
        text-decoration: none;

        &:hover {
          background: var(--gray-100);
          border-color: var(--gray-400);
          color: var(--gray-900);
        }

        &:focus {
          box-shadow: none;
          background: var(--gray-100);
          outline: none;
        }
      }

      &.active .page-link {
        background: var(--gray-900);
        border-color: var(--gray-900);
        color: var(--cream);
      }

      &.disabled .page-link {
        background: var(--gray-100);
        color: var(--gray-400);
        cursor: not-allowed;
        pointer-events: none;
      }

      &:first-child .page-link {
        border-top-left-radius: var(--border-radius);
        border-bottom-left-radius: var(--border-radius);
      }

      &:last-child .page-link {
        border-top-right-radius: var(--border-radius);
        border-bottom-right-radius: var(--border-radius);
      }
    }

    &.pagination-sm {
      .page-item .page-link {
        min-width: 2rem;
        height: 2rem;
        font-size: 0.8rem;
        padding: 0 0.5rem;
      }
    }
  }
</style>
