# urlFetch
Con-current url fetch and search

Input: a list of url's from a file; one per line starting from the second line.
    1, "facebook.com/"
    ...
    n,"pcworld.com/"
Output: a file
    1, "facebook.com/": match1, match2,...
    ...
    n,"pcworld.com/": match1, match2,...
If the HTTP request returns error, just record the error.

urlFetchAndSearch is a con-current program that fetchs the content of the home page of these url's and find all matches (a regex, case insensitive) with the constraint that there can't be more than 20 HTTP requests.

