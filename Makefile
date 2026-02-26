# Default target
.DEFAULT_GOAL := help

MAGE := go tool mage

# .PHONY targets
.PHONY: watch
# Watch mode
watch:
	OPEN_BROWSER=false $(MAGE) watch

# Help target to display available targets and their descriptions
help:
	@echo "Available targets:"
	@echo ""
	@echo "  watch              - Run templ, air, and tailwind in watch mode without opening browser"
