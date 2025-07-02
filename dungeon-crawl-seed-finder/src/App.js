import React, { useState, useEffect, useRef } from 'react';
import HCaptcha from '@hcaptcha/react-hcaptcha';
import crawlVersions from './versions';
import './App.css';

function App() {
    const [results, setResults] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [resultsUrl, setResultsUrl] = useState(null);
    const [captchaToken, setCaptchaToken] = useState(null);
    const [showTooltip, setShowTooltip] = useState(false);
    const captchaRef = useRef(null);
    const pollCountRef = useRef(0);

    useEffect(() => {
        const fetchResults = async () => {
            if (resultsUrl) {
                if (pollCountRef.current >= 20) {
                    setIsLoading(false);
                    return;
                }
                try {
                    pollCountRef.current += 1;
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
        if (!captchaToken) {
            alert('Please complete the captcha first');
            return;
        }

        setIsLoading(true);
        setResults('');
        setResultsUrl(null);
        pollCountRef.current = 0;

        const formData = new FormData(e.target);
        const urlEncodedData = new URLSearchParams();
        urlEncodedData.append('crawl_version', formData.get('crawl_version'));
        urlEncodedData.append('regexp', formData.get('regexp'));
        urlEncodedData.append('attempts', formData.get('attempts'));
        urlEncodedData.append('depth', formData.get('depth'));
        urlEncodedData.append('h-captcha-response', captchaToken);

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
            // Reset the captcha after submission
            if (captchaRef.current) {
                captchaRef.current.resetCaptcha();
            }
            setCaptchaToken(null);
        }
    };

    const handleVerificationSuccess = (token) => {
        setCaptchaToken(token);
    };

    const renderResults = () => {
        if (!results) return null;
        
        try {
            const parsedResults = JSON.parse(results);
            if (!Array.isArray(parsedResults)) {
                return <div>Invalid results format</div>;
            }

            return (
                <table className="results-table">
                    <thead>
                        <tr>
                            <th>Host</th>
                            <th>Seed</th>
                            <th>IPFS Hash</th>
                            <th>Status</th>
                        </tr>
                    </thead>
                    <tbody>
                        {parsedResults.map((result, index) => (
                            <tr key={index}>
                                <td>{result.host}</td>
                                <td>{result.status !== 'gave up' ? result.seed : ''}</td>
                                <td>
                                    <a href={`/result?resulthash=${result.ipfshash}`}>
                                        {result.ipfshash}
                                    </a>
                                </td>
                                <td>{result.status}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            );
        } catch (error) {
            return <div>Error parsing results: {error.message}</div>;
        }
    };

    return (
        <div className="App">
            <h1>Dungeon Crawl Stone Soup Seed Finder</h1>
            <h2>Choose some constraints below and search for a seed that matches.</h2>
            <form onSubmit={handleSubmit}>
                <div className="content">
                    {/* Version */}
                    <div className="colContainer">
                        <div className="col">Version</div>
                        <div className="col">
                            <select name="crawl_version">
                                {crawlVersions.map(version => (
                                    <option key={version} value={version}>{version}</option>
                                ))}
                            </select>
                        </div>
                    </div>

                    {/* Regexp */}
                    <div className="colContainer">
                        <div className="col">
                            <span 
                                className="tooltip-container"
                                onMouseEnter={() => setShowTooltip(true)}
                                onMouseLeave={() => setShowTooltip(false)}
                            >
                                Regexp
                            </span>
                        </div>
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

                    {/* hCaptcha */}
                    <div className="colContainer">
                        <div className="col">
                            <HCaptcha
                                sitekey="491410d4-5e9c-43dc-a27c-64f86e22a6cc"
				theme="dark"
                                onVerify={handleVerificationSuccess}
                                ref={captchaRef}
                            />
                        </div>
                        <div className="col"></div>
                    </div>

                    {/* Submit Button */}
                    <div className="colContainer">
                        <div className="col">
                            <input type="submit" value="Submit" />
                        </div>
                        <div className="col"></div>
                    </div>
                </div>
            </form>

            {/* Results Section */}
            {isLoading && <div>Publishing request...</div>}
            {results && (
                <div className="results">
                    {renderResults()}
                </div>
            )}

            {/* Fixed position tooltip */}
            {showTooltip && (
                <div className="tooltip-overlay">
			Example: 'quick blade', 'broad axe' or combining items '(?s)dagger of drain.*?orb of guile'
                </div>
            )}
        </div>
    );
}

export default App;

