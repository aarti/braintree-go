package braintree

import (
	"context"
	"encoding/xml"
	"net/url"
	"time"
)

type CreditCardGateway struct {
	*Braintree
}

// Create creates a new credit card.
func (g *CreditCardGateway) Create(ctx context.Context, card *CreditCard) (*CreditCard, error) {
	resp, err := g.execute(ctx, "POST", "payment_methods", card)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case 201:
		return resp.creditCard()
	}
	return nil, &invalidResponseError{resp}
}

// Update updates a credit card.
func (g *CreditCardGateway) Update(ctx context.Context, card *CreditCard) (*CreditCard, error) {
	resp, err := g.execute(ctx, "PUT", "payment_methods/"+card.Token, card)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case 200:
		return resp.creditCard()
	}
	return nil, &invalidResponseError{resp}
}

// Find finds a credit card by payment method token.
func (g *CreditCardGateway) Find(ctx context.Context, token string) (*CreditCard, error) {
	resp, err := g.execute(ctx, "GET", "payment_methods/"+token, nil)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case 200:
		return resp.creditCard()
	}
	return nil, &invalidResponseError{resp}
}

// Delete deletes a credit card.
func (g *CreditCardGateway) Delete(ctx context.Context, card *CreditCard) error {
	resp, err := g.execute(ctx, "DELETE", "payment_methods/"+card.Token, nil)
	if err != nil {
		return err
	}
	switch resp.StatusCode {
	case 200:
		return nil
	}
	return &invalidResponseError{resp}
}

// ExpiringBetweenIDs finds IDs of credit cards that expire between the
// specified dates, returning the IDs only. Use ExpiringBetween and
// ExpiringBetweenNext to get pages of credit cards.
func (g *CreditCardGateway) ExpiringBetweenIDs(ctx context.Context, fromDate, toDate time.Time) (*SearchResult, error) {
	qs := url.Values{}
	qs.Set("start", fromDate.UTC().Format("012006"))
	qs.Set("end", toDate.UTC().Format("012006"))
	resp, err := g.execute(ctx, "POST", "/payment_methods/all/expiring_ids?"+qs.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var searchResult struct {
		PageSize int `xml:"page-size"`
		Ids      struct {
			Item []string `xml:"item"`
		} `xml:"ids"`
	}
	err = xml.Unmarshal(resp.Body, &searchResult)
	if err != nil {
		return nil, err
	}

	return &SearchResult{
		PageSize: searchResult.PageSize,
		IDs:      searchResult.Ids.Item,
	}, nil
}

// ExpiringBetweenPage finds credit cards that expire between the specified
// dates, returning a specific page of results. Page numbers start at 1.
func (g *CreditCardGateway) ExpiringBetweenPage(ctx context.Context, fromDate, toDate time.Time, searchResult *SearchResult, page int) (*CreditCardSearchResult, error) {
	startOffset := (page - 1) * searchResult.PageSize
	endOffset := startOffset + searchResult.PageSize
	if endOffset > len(searchResult.IDs) {
		endOffset = len(searchResult.IDs)
	}
	if startOffset >= endOffset {
		return nil, nil
	}

	query := &SearchQuery{}
	query.AddMultiField("ids").Items = searchResult.IDs[startOffset:endOffset]
	creditCards, err := g.fetchExpiringBetween(ctx, query, fromDate, toDate)

	result := &CreditCardSearchResult{
		TotalItems:        len(searchResult.IDs),
		TotalIDs:          searchResult.IDs,
		CurrentPageNumber: page,
		PageSize:          searchResult.PageSize,
		CreditCards:       creditCards,
	}

	return result, err
}

func (g *CreditCardGateway) fetchExpiringBetween(ctx context.Context, query *SearchQuery, fromDate, toDate time.Time) ([]*CreditCard, error) {
	qs := url.Values{}
	qs.Set("start", fromDate.UTC().Format("012006"))
	qs.Set("end", toDate.UTC().Format("012006"))
	resp, err := g.execute(ctx, "POST", "/payment_methods/all/expiring?"+qs.Encode(), query)
	if err != nil {
		return nil, err
	}

	var v struct {
		CreditCards []*CreditCard `xml:"credit-card"`
	}

	err = xml.Unmarshal(resp.Body, &v)
	if err != nil {
		return nil, err
	}

	return v.CreditCards, nil
}
