// Core Components (existing)
export { default as Button } from "./components/Button.svelte";
export { default as Modal } from "./components/Modal.svelte";
export { default as Tile } from "./components/Tile.svelte";
export { default as Breadcrumb } from "./components/Breadcrumb.svelte";
export { default as Navbar } from "./components/Navbar.svelte";
export { default as Flashcard } from "./components/Flashcard.svelte";
export { default as Highlighter } from "./components/Highlighter.svelte";
export { default as CitationPopup } from "./components/CitationPopup.svelte";
export { default as EditModal } from "./components/EditModal.svelte";
export { default as ConfirmModal } from "./components/ConfirmModal.svelte";
export { default as Pagination } from "./components/Pagination.svelte";
export { default as StatusIndicator } from "./components/StatusIndicator.svelte";
export { default as NotificationBanner } from "./components/NotificationBanner.svelte";
export { default as ExportMenu } from "./components/ExportMenu.svelte";

// Layout Components
export { default as PageLayout } from "./components/layout/PageLayout.svelte";
export { default as Section } from "./components/layout/Section.svelte";
export { default as PageHeader } from "./components/layout/PageHeader.svelte";
export { default as CardContainer } from "./components/layout/CardContainer.svelte";

// Form Components
export { default as FormLabel } from "./components/forms/FormLabel.svelte";
export { default as TextInput } from "./components/forms/TextInput.svelte";
export { default as TextArea } from "./components/forms/TextArea.svelte";
export { default as SelectInput } from "./components/forms/SelectInput.svelte";
export { default as FormField } from "./components/forms/FormField.svelte";

// Feedback Components
export { default as EmptyState } from "./components/feedback/EmptyState.svelte";
export { default as LoadingState } from "./components/feedback/LoadingState.svelte";
export { default as ErrorState } from "./components/feedback/ErrorState.svelte";

// Navigation Components
export { default as NavCard } from "./components/navigation/NavCard.svelte";
export { default as ActionTile } from "./components/navigation/ActionTile.svelte";
export { default as VerticalTileList } from "./components/navigation/VerticalTileList.svelte";

// Utility Components
export { default as IconWrapper } from "./components/utils/IconWrapper.svelte";
export { default as CostBadge } from "./components/utils/CostBadge.svelte";
export { default as TextWithHighlight } from "./components/utils/TextWithHighlight.svelte";
export { default as MetaBadge } from "./components/utils/MetaBadge.svelte";

// Composite Components
export { default as TileGrid } from "./components/composite/TileGrid.svelte";
export { default as WorkspaceSection } from "./components/composite/WorkspaceSection.svelte";
export { default as ContentBlock } from "./components/composite/ContentBlock.svelte";

// Utilities
export { getLanguageName, capitalize, formatActivityType } from "./utils";
