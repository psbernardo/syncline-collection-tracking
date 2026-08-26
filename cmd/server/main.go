package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/config"
	"github.com/psbernardo/syncline-collection-tracking/internal/platform/database"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/accounts"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/dashboard"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/invoices"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/products"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/purchaseorders"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/quotations"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/receivables"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/salesorders"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/suppliers"
	webstatic "github.com/psbernardo/syncline-collection-tracking/internal/web/static"
	webtemplates "github.com/psbernardo/syncline-collection-tracking/internal/web/templates"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, closeDB, err := database.Open(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeDB()

	dashboardRepository := dashboard.NewGormRepository(db)
	accountHandler, err := accounts.NewHandler(accounts.NewService(db, accounts.NewGormRepository(db)), dashboard.NewCompanyTotalsService(dashboardRepository))
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	accountHandler.RegisterRoutes(mux)
	productHandler, err := products.NewHandler(products.NewService(db, products.NewGormRepository(db)))
	if err != nil {
		return err
	}
	productHandler.RegisterRoutes(mux)
	productRepository := products.NewGormRepository(db)
	accountRepository := accounts.NewGormRepository(db)
	invoiceRepository := invoices.NewGormRepository(db)
	supplierRepository := suppliers.NewGormRepository(db)
	productOptions := func(ctx context.Context) ([]suppliers.ProductOption, error) {
		items, err := productRepository.List(ctx, true)
		if err != nil {
			return nil, err
		}
		options := make([]suppliers.ProductOption, 0, len(items))
		for _, item := range items {
			options = append(options, suppliers.ProductOption{ID: item.ID, SKU: item.SKU, Name: item.Name, UOM: item.UOM})
		}
		return options, nil
	}
	accountOptions := webtemplates.OptionsProvider(func(ctx context.Context) ([]webtemplates.SearchableSelectOption, error) {
		items, err := accountRepository.List(ctx)
		if err != nil {
			return nil, err
		}
		options := make([]webtemplates.SearchableSelectOption, 0, len(items))
		for _, item := range items {
			options = append(options, webtemplates.SearchableSelectOption{Value: fmt.Sprintf("%d", item.ID), Label: item.CompanyName, Search: item.CompanyName})
		}
		return options, nil
	})
	searchableProductOptions := webtemplates.OptionsProvider(func(ctx context.Context) ([]webtemplates.SearchableSelectOption, error) {
		items, err := productRepository.List(ctx, true)
		if err != nil {
			return nil, err
		}
		options := make([]webtemplates.SearchableSelectOption, 0, len(items))
		for _, item := range items {
			label := item.SKU + " - " + item.Name + " (" + item.UOM + ")"
			options = append(options, webtemplates.SearchableSelectOption{Value: fmt.Sprintf("%d", item.ID), Label: label, Description: item.Description, Search: strings.ToLower(label + " " + item.Description), UOM: item.UOM})
		}
		return options, nil
	})
	customerOptions := salesorders.CustomerOptions(func(ctx context.Context) ([]salesorders.CustomerOption, error) {
		items, err := accountRepository.List(ctx)
		if err != nil {
			return nil, err
		}
		options := make([]salesorders.CustomerOption, 0, len(items))
		for _, item := range items {
			options = append(options, salesorders.CustomerOption{ID: item.ID, Name: item.CompanyName})
		}
		return options, nil
	})
	orderProductOptions := salesorders.ProductOptions(func(ctx context.Context) ([]salesorders.ProductOption, error) {
		items, err := productRepository.List(ctx, true)
		if err != nil {
			return nil, err
		}
		options := make([]salesorders.ProductOption, 0, len(items))
		for _, item := range items {
			options = append(options, salesorders.ProductOption{ID: item.ID, SKU: item.SKU, Name: item.Name, UOM: item.UOM})
		}
		return options, nil
	})
	supplierHandler, err := suppliers.NewHandler(suppliers.NewService(db, supplierRepository, productOptions))
	if err != nil {
		return err
	}
	supplierHandler.RegisterRoutes(mux)
	receivableHandler, err := receivables.NewHandler(receivables.NewService(db, receivables.NewGormRepository(db), accountRepository, receivables.NewGormInvoiceRepository(db)))
	if err != nil {
		return err
	}
	receivableHandler.RegisterRoutes(mux)
	quotationRepository := quotations.NewGormRepository(db)
	salesOrderRepository := salesorders.NewGormRepository(db)
	quotationHandler, err := quotations.NewHandler(quotationRepository, accountOptions, searchableProductOptions, salesOrderRepository)
	if err != nil {
		return err
	}
	quotationHandler.RegisterRoutes(mux)
	salesOrderHandler := salesorders.NewHandler(salesOrderRepository, quotationRepository, customerOptions, orderProductOptions)
	salesOrderHandler.RegisterRoutes(mux)
	purchaseSupplierOptions := purchaseorders.SupplierOptions(func(ctx context.Context) ([]purchaseorders.SupplierOption, error) {
		items, err := supplierRepository.List(ctx, true)
		if err != nil {
			return nil, err
		}
		options := make([]purchaseorders.SupplierOption, 0, len(items))
		for _, item := range items {
			options = append(options, purchaseorders.SupplierOption{ID: item.ID, Name: item.Name})
		}
		return options, nil
	})
	purchaseOrderRepository := purchaseorders.NewGormRepository(db)
	purchaseProductOptions := purchaseorders.ProductOptions(func(ctx context.Context) ([]purchaseorders.ProductOption, error) {
		items, err := productRepository.List(ctx, true)
		if err != nil {
			return nil, err
		}
		options := make([]purchaseorders.ProductOption, 0, len(items))
		for _, item := range items {
			options = append(options, purchaseorders.ProductOption{ID: item.ID, SKU: item.SKU, Name: item.Name, UOM: item.UOM})
		}
		return options, nil
	})
	purchaseOrderHandler, err := purchaseorders.NewHandler(salesOrderRepository, purchaseOrderRepository, purchaseSupplierOptions, purchaseOrderRepository, purchaseProductOptions)
	if err != nil {
		return err
	}
	purchaseOrderHandler.RegisterRoutes(mux)
	invoiceHandler := invoices.NewHandler(salesOrderRepository, invoiceRepository)
	invoiceHandler.RegisterRoutes(mux)
	dashboardHandler, err := dashboard.NewHandler(dashboard.NewService(dashboardRepository))
	if err != nil {
		return err
	}
	dashboardHandler.RegisterRoutes(mux)
	dashboardHandler.RegisterSalesRoutes(mux, dashboard.NewSalesService(dashboardRepository))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(webstatic.FS))))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/accounts", http.StatusSeeOther)
	})

	server := &http.Server{
		Addr:              fmt.Sprintf("0.0.0.0:%d", cfg.HTTPPort),
		Handler:           requestTimeouts(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	fmt.Printf("collection tracking listening on http://%s\n", server.Addr)
	return server.ListenAndServe()
}

func requestTimeouts(next http.Handler) http.Handler {
	return http.TimeoutHandler(next, 15*time.Second, "request timed out")
}
