public class RecursiveDS {
    // Singly linked list node
    static class Node {
        int val;
        Node next;
        Node(int val, Node next) {
            this.val = val;
            this.next = next;
        }
    }

    // Build list: 1 -> 2 -> 3 -> null
    static Node buildList() {
        return new Node(1, new Node(2, new Node(3, null)));
    }

    // Sum all elements
    static int sum(Node n) {
        int s = 0;
        while (n != null) {
            s += n.val;
            n = n.next;
        }
        return s;
    }

    // Reverse the list
    static Node reverse(Node head) {
        Node prev = null;
        Node curr = head;
        while (curr != null) {
            Node next = curr.next;
            curr.next = prev;
            prev = curr;
            curr = next;
        }
        return prev;
    }

    // Print list
    static void printList(Node n) {
        while (n != null) {
            System.out.println(n.val);
            n = n.next;
        }
    }

    public static void main(String[] args) {
        Node list = buildList();
        System.out.println(sum(list));  // 6

        Node rev = reverse(list);
        printList(rev);                 // 3 2 1

        // Recursive factorial
        System.out.println(factorial(10)); // 3628800
    }

    static int factorial(int n) {
        if (n <= 1) return 1;
        return n * factorial(n - 1);
    }
}
