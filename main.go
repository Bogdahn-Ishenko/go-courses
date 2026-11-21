package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type Film struct {
	ID       int
	Title    string
	Year     int
	Director string
}

type Result struct {
	Film *Film
	Err  error
}

func main() {
	fmt.Println("id — title — year — director")
	fromPtr := flag.Int("from", -1, "ID первого фильма в списке")
	toPtr := flag.Int("to", -1, "ID последнего фильма в списке")
	workersPtr := flag.Int("workers", 10, "Количество воркеров")
	timeoutPtr := flag.Int("timeout", 5, "Таймаут в секундах")

	flag.Parse()

	var client Client

	if *fromPtr < 0 || *toPtr < 0 {
		err := fmt.Errorf("Не заполнены ID первого и последнего фильма")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *fromPtr > *toPtr {
		err := fmt.Errorf("Некорректный диапазон")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *workersPtr <= 0 {
		err := fmt.Errorf("Некорректное число воркеров")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *timeoutPtr <= 0 {
		err := fmt.Errorf("Некорректный таймаут")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	results := make(chan Result, *toPtr-*fromPtr+1)
	jobs := make(chan int, *toPtr-*fromPtr+1)

	baseURL := os.Getenv("MOVIE_BASE_URL")
	if baseURL != "" {
		client.baseURL = baseURL
		
	} else {
		client.baseURL = "https://homeworksite.site"
	}

	client.httpClient = &http.Client{
		Timeout: time.Duration(*timeoutPtr) * time.Second,
	}

	var wg sync.WaitGroup

	for w := 1; w <= *workersPtr; w++ {
		wg.Add(1)
		go worker(w, jobs, results, client, ctx, &wg)
	}

	for filmID := *fromPtr; filmID <= *toPtr; filmID++ {
		jobs <- filmID
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		if res.Err != nil {
			fmt.Println(res.Err)
			continue
		}
		fmt.Printf("%d — %s — %d — %s\n", res.Film.ID, res.Film.Title, res.Film.Year, res.Film.Director)
	}
}

func worker(id int, jobs <-chan int, results chan<- Result, client Client, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for j := range jobs {
		if ctx.Err() != nil {
			return
		}
		var result Result
		film, err := client.listFilms(j, ctx)
		if err != nil {
			result.Err = fmt.Errorf("Ошибка при получении информации о фильме №%d\n%s", j, err)
			results <- result
			continue
		}
		result.Film = film
		results <- result
	}
}

func (c *Client) listFilms(filmID int, ctx context.Context) (*Film, error) {
	url := fmt.Sprintf("%s/%d/info.0.json", c.baseURL, filmID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("Ошибка при формировании контекста %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Ошибка при выполнении запроса: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, err := io.ReadAll(resp.Body)

		if err != nil {
			return nil, fmt.Errorf("Ошибка при чтении тела ответа: %s\n %v", resp.Status, err)
		}

		return nil, fmt.Errorf("Ошибка при выполнении запроса: %s\n %v", resp.Status, string(msg))
	}

	var film Film

	if err := json.NewDecoder(resp.Body).Decode(&film); err != nil {
		return nil, fmt.Errorf("Ошибка при декодировании JSON: %v", err)
	}
	return &film, nil
}
