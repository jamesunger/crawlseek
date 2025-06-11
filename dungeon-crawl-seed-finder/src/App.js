import React, { useState, useEffect } from 'react';
import './App.css';

function Navbar() {
    return (
        <nav className="navbar">
            <div className="logo">
                <a href="/">crawlseek</a>
            </div>
        </nav>
    );
}

function App() {
    const [results, setResults] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [resultsUrl, setResultsUrl] = useState(null);

    useEffect(() => {
        const fetchResults = async () => {
            if (resultsUrl) {
                try {
                    const response = await fetch(resultsUrl);
                    const result = await response.text();
                    setResults(result);
		    setIsLoading(false);
                } catch (error) {
                    setResults('Error fetching results: ' + error.message);
		    setIsLoading(false);
                }
            }
        };

        if (resultsUrl) {
            const interval = setInterval(fetchResults, 5000); // Poll every 5 second
            return () => clearInterval(interval);
        }
    }, [resultsUrl]);

    const handleSubmit = async (e) => {
        e.preventDefault();
        setIsLoading(true);
        setResults('');
        setResultsUrl(null);

        const formData = new FormData(e.target);
        const urlEncodedData = new URLSearchParams();
        urlEncodedData.append('crawl_version', formData.get('crawl_version'));
        urlEncodedData.append('regexp', formData.get('regexp'));
        urlEncodedData.append('attempts', formData.get('attempts'));
        urlEncodedData.append('depth', formData.get('depth'));

        try {
            const response = await fetch('/enqueue', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: urlEncodedData.toString()
            });

            const result = await response.json();
            if (result.results_url) {
                setResultsUrl(result.results_url);
            } else {
                setResults('Error: No results URL in response');
            }
        } catch (error) {
            setResults('Error: ' + error.message);
        } finally {
            setIsLoading(true);
        }
    };

    return (
        <div className="App">
            <Navbar />
            <h1>Dungeon Crawl Stone Soup Seed Finder</h1>
            <h2>Choose some constraints below and search for a seed that matches.</h2>
            <form onSubmit={handleSubmit}>
                <div className="content">
                    {/* Version */}
                    <div className="colContainer">
                        <div className="col">Version</div>
                        <div className="col">
                            <select name="crawl_version">
                                <option value="trunk">trunk</option>
                                <option value="0.24.1">0.24.1</option>
                                <option value="0.25.1">0.25.1</option>
                                <option value="0.26.1">0.26.1</option>
                                <option value="0.28.0">0.28.0</option>
                                <option value="0.29.1">0.29.1</option>
                                <option value="0.30.2">0.30.2</option>
                                <option value="0.31.2">0.31.2</option>
                                <option value="0.32.1">0.32.1</option>
                                <option value="0.33.0">0.33.0</option>
                            </select>
                        </div>
                    </div>

                    {/* Regexp */}
                    <div className="colContainer">
                        <div className="col">Regexp</div>
                        <div className="col">
                            <input type="text" name="regexp" />
                        </div>
                    </div>

                    {/* Attempts */}
                    <div className="colContainer">
                        <div className="col">Attempts</div>
                        <div className="col">
                            <select name="attempts">
                                <option>5</option>
                                <option>10</option>
                                <option>15</option>
                                <option>20</option>
                            </select>
                        </div>
                    </div>

                    {/* Depth */}
                    <div className="colContainer">
                        <div className="col">Depth</div>
                        <div className="col">
                            <select name="depth">
                                <option>5</option>
                                <option>10</option>
                                <option>15</option>
                            </select>
                        </div>
                    </div>

                    {/* Submit Button */}
                    <div className="colContainer">
                        <div className="col">
                            <input type="submit" value="Submit" />
                        </div>
                    </div>
                </div>
            </form>

            {/* Results Section */}
            {isLoading && <div>Publishing request...</div>}
            {results && (
                <div className="results">
                    <h3>Results:</h3>
                    {results}
                </div>
            )}
        </div>
    );
}

export default App;

